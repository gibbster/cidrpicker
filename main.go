package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sort"

	"github.com/DavidGamba/go-getoptions"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var Logger = log.New(os.Stderr, "", log.LstdFlags)

func main() {
	os.Exit(program(os.Args))
}

func program(args []string) int {
	opt := getoptions.New()
	opt.Self(filepath.Base(os.Args[0]), "Return the next available CIDR block of the given size for a given VPC.")
	opt.Bool("quiet", false)

	list := opt.NewCommand("list", "List VPCs").SetCommandFn(ListRun)
	list.SetUnknownMode(getoptions.Pass)
	list.String("profile", "", opt.ArgName("aws_profile"))
	list.String("region", "", opt.ArgName("aws_region"))
	list.String("vpc-id", "")
	free := opt.NewCommand("free", "Find the next free CIDR block for the given size.").SetCommandFn(FreeRun)
	free.String("profile", "", opt.ArgName("aws_profile"))
	free.String("region", "", opt.ArgName("aws_region"))
	free.String("vpc-id", "", opt.Required())
	free.Int("size", 24)
	free.Int("number", 1, opt.Description("number of CIDR blocks to generate."))
	opt.HelpCommand("help", opt.Alias("?"))
	remaining, err := opt.Parse(args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}
	if opt.Called("quiet") {
		Logger.SetOutput(io.Discard)
	}

	ctx, cancel, done := getoptions.InterruptContext()
	defer func() { cancel(); <-done }()

	err = opt.Dispatch(ctx, remaining)
	if err != nil {
		if errors.Is(err, getoptions.ErrorHelpCalled) {
			return 1
		}
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}
	return 0
}

type netSubnet struct {
	Subnet types.Subnet
	Net    net.IPNet
}

func ListRun(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	profile := opt.Value("profile").(string)
	region := opt.Value("region").(string)
	vpcID := opt.Value("vpc-id").(string)
	Logger.Printf("profile: %s, region: %s, vpcID: %s", profile, region, vpcID)

	cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile), config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("unable to load SDK config: %w", err)
	}
	client := ec2.NewFromConfig(cfg)

	input := &ec2.DescribeVpcsInput{}

	if vpcID != "" {
		input.VpcIds = []string{vpcID}
	}

	output, err := client.DescribeVpcs(ctx, input)
	if err != nil {
		return err
	}
	for _, vpc := range output.Vpcs {
		fmt.Printf("VPC: %-21s, CIDR: %s\n", *vpc.VpcId, *vpc.CidrBlock)

		subnets, err := getVPCSubnets(ctx, client, *vpc.VpcId)
		if err != nil {
			return err
		}

		netSubnets := []netSubnet{}
		for _, subnet := range subnets {
			_, c, err := net.ParseCIDR(*subnet.CidrBlock)
			if err != nil {
				return err
			}
			netSubnets = append(netSubnets, netSubnet{subnet, *c})
		}

		sort.Slice(netSubnets, func(i, j int) bool {
			return bytes.Compare(netSubnets[i].Net.IP, netSubnets[j].Net.IP) < 0
		})
		for _, subnet := range netSubnets {
			Logger.Printf("Subnet %-24s %s %s\n", *subnet.Subnet.SubnetId, *subnet.Subnet.AvailabilityZone, subnet.Net.String())
		}
	}

	return nil
}

func FreeRun(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	profile := opt.Value("profile").(string)
	region := opt.Value("region").(string)
	vpcID := opt.Value("vpc-id").(string)
	size := opt.Value("size").(int)
	number := opt.Value("number").(int)
	Logger.Printf("profile: %s, region: %s, vpcID: %s, size: %d", profile, region, vpcID, size)

	cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile), config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := ec2.NewFromConfig(cfg)

	vpc, err := getVPC(ctx, client, vpcID)
	if err != nil {
		return err
	}
	Logger.Printf("VPC %s %s\n", *vpc.VpcId, *vpc.CidrBlock)

	subnets, err := getVPCSubnets(ctx, client, vpcID)
	if err != nil {
		return err
	}

	netSubnets := []netSubnet{}
	netCIDRs := []net.IPNet{}
	for _, subnet := range subnets {
		_, c, err := net.ParseCIDR(*subnet.CidrBlock)
		if err != nil {
			return err
		}
		netSubnets = append(netSubnets, netSubnet{subnet, *c})
		netCIDRs = append(netCIDRs, *c)
	}

	sort.Slice(netSubnets, func(i, j int) bool {
		return bytes.Compare(netSubnets[i].Net.IP, netSubnets[j].Net.IP) < 0
	})
	for _, subnet := range netSubnets {
		Logger.Printf("Subnet %-24s %s %s\n", *subnet.Subnet.SubnetId, *subnet.Subnet.AvailabilityZone, subnet.Net.String())
	}

	_, vpcNetCIDR, err := net.ParseCIDR(*vpc.CidrBlock)
	if err != nil {
		return err
	}

	for i := 0; i < number; i++ {
		result := FindBlock(*vpcNetCIDR, &netCIDRs, size)
		fmt.Printf("%s\n", result.String())
		netCIDRs = append(netCIDRs, result)
	}

	return nil
}

func getVPC(ctx context.Context, client *ec2.Client, vpcID string) (types.Vpc, error) {
	input := &ec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	}
	output, err := client.DescribeVpcs(ctx, input)
	if err != nil {
		return types.Vpc{}, err
	}
	if len(output.Vpcs) < 1 {
		return types.Vpc{}, fmt.Errorf("not found")
	}
	return output.Vpcs[0], nil
}

func getVPCSubnets(ctx context.Context, client *ec2.Client, vpcID string) ([]types.Subnet, error) {
	input := &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{{
			Name:   aws.String("vpc-id"),
			Values: []string{vpcID},
		}},
	}
	output, err := client.DescribeSubnets(ctx, input)
	return output.Subnets, err
}

func FindBlock(vpc net.IPNet, occupied *[]net.IPNet, size int) (result net.IPNet) {
	blocks := []net.IPNet{vpc}
	for len(blocks) > 0 {
		//pop the first element off the queue
		block := blocks[0]
		blocks = blocks[1:]
		if netInList(&block, occupied) {
			continue
		}

		if cidrSize(block) == size {
			if netContainsNets(&block, occupied) {
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

func netInList(net *net.IPNet, list *[]net.IPNet) bool {
	found := false
	for _, test := range *list {
		if net.String() == test.String() {
			found = true
		}
	}
	return found
}

func cidrSize(net net.IPNet) int {
	size, _ := net.Mask.Size()
	return size
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
