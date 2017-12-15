package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"log"
	"net"
)

func getVPCCidrBlock(sess *session.Session, vpcId string) (string, error) {
	svc := ec2.New(sess)

	input := &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					aws.String(vpcId),
				},
			},
		},
	}

	result, err := svc.DescribeVpcs(input)
	if err != nil {
		return "", err
	}

	return *result.Vpcs[0].CidrBlock, nil
}

func getVPCSubnetCidrBlocks(sess *session.Session, vpcId string) ([]string, error) {
	svc := ec2.New(sess)
	input := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					aws.String(vpcId),
				},
			},
		},
	}

	result, err := svc.DescribeSubnets(input)
	if err != nil {
		return nil, err
	}

	blocks := make([]string, len(result.Subnets))
	for i, subnet := range result.Subnets {
		blocks[i] = *subnet.CidrBlock
	}
	return blocks, nil
}

func handleAwsErr(err error) {
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		default:
			log.Fatal(aerr.Error())
		}
	} else {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		log.Fatal(err.Error())
	}
	return
}

func bisectSubnet(subnet net.IPNet) (subnetA net.IPNet, subnetB net.IPNet) {
	mask_size, total_size := subnet.Mask.Size()
	var bit_to_switch byte = byte(total_size - mask_size - 1)

	subnetA.Mask = make([]byte, 4)
	copy(subnetA.Mask, subnet.Mask)
	subnetA.Mask[3-(bit_to_switch/8)] |= (1 << (bit_to_switch % 8))

	subnetA.IP = make([]byte, 4)
	copy(subnetA.IP, subnet.IP)

	subnetB.Mask = make([]byte, 4)
	copy(subnetB.Mask, subnet.Mask)
	subnetB.Mask[3-(bit_to_switch/8)] |= (1 << (bit_to_switch % 8))

	subnetB.IP = make([]byte, 4)
	copy(subnetB.IP, subnet.IP)
	subnetB.IP[3-(bit_to_switch/8)] |= (1 << (bit_to_switch % 8))

	return
}

func cidrSize(net net.IPNet) int {
	size, _ := net.Mask.Size()
	return size
}

func netInList(net *net.IPNet, list *[]net.IPNet) bool {
	found := false
	//log.Printf("Checking against %v\n", net.String())
	for _, test := range *list {
		//log.Printf("Testing %v\n", test.String())
		if net.String() == test.String() {
			found = true
		}
	}
	return found
}

func netContainsNets(net *net.IPNet, list *[]net.IPNet) bool {
	contains := false
	for _, test := range *list {
		if net.Contains(test.IP) {
			contains = true
			break
		}
	}
	return contains
}

func FindBlock(vpc net.IPNet, occupied *[]net.IPNet, size int) (result net.IPNet) {
	blocks := []net.IPNet{vpc}
	//log.Printf("Looking for a block of size %v\n", size)
	for len(blocks) > 0 {
		//pop the first element off the queue
		//log.Printf("blocks are set to %v\n", blocks)
		block := blocks[0]
		blocks = blocks[1:]
		//block := blocks[len(blocks)-1]
		//blocks = blocks[:len(blocks)-1]
		if netInList(&block, occupied) {
			//log.Printf("Found block %v in occupied list :-(", block.String())
			continue
		}

		//log.Printf("block is size %v\n", cidrSize(block))
		if cidrSize(block) == size {
			//log.Println("Block is the right size")
			if netContainsNets(&block, occupied) {
				//log.Printf("Found one of the occupied nets within %v :-(", block.String())
				continue
			} else {
				result = block
				break
			}
		} else if cidrSize(block) < size { 
			left, right := bisectSubnet(block)
			blocks = append(blocks, left)
			blocks = append(blocks, right)
			continue
		}
	}
	return
}

func main() {
	vpcId := flag.String("vpcid", "", "VPC ID to use")
	size := flag.Int("size", 24, "Size of CIDR block to find")
	flag.Parse()

	if *vpcId == "" {
		log.Fatal("Must specify a VPC ID")
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	vpcCidrBlock, err := getVPCCidrBlock(sess, *vpcId)
	if err != nil {
		handleAwsErr(err)
		return
	}
	subnetCidrs, err := getVPCSubnetCidrBlocks(sess, *vpcId)
	if err != nil {
		handleAwsErr(err)
		return
	}
	_, vpc, err := net.ParseCIDR(vpcCidrBlock)

	if err != nil {
		log.Fatal("error, %v\n", err)
	} else {
		subnets := make([]net.IPNet, len(subnetCidrs))
		for i, cidr := range subnetCidrs {
			_, sn, err := net.ParseCIDR(cidr)
			if err != nil {
				log.Fatal("error, %v\n", err)
			}
			subnets[i] = *sn
		}
		block := FindBlock(*vpc, &subnets, *size)
		//fmt.Printf("VPC: %v\n", vpc)
		//fmt.Printf("subnets: %v\n", subnetCidrs)
		//fmt.Printf("found this block: %v\n", block.String())
		fmt.Println(block.String())
	}
}
