# CIDR Picker

Checks to see if there's enough space in a VPC for a subnet of a given size. If there is, return it in the form of a CIDR block.

## Getting Started

Clone the repository
`git clone git@github.com:gibbster/cidrpicker.git`

Build and test
`go build`
`go test`

Configure the AWS credentials through `~./aws/config`, environment variables, or AWS IAM Role (see https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html)

Get help:
```
./cidrpicker -h                                                                                                                                                                                                      git:master*
Usage of ./cidrpicker:
  -size int
    	Size of CIDR block to find (default 24)
  -vpcid string
    	VPC ID to use
```

Run the program
```
./cidrpicker -vpicid vpc-XXXXXXX -size 24
10.0.4.0/23
```

## TODO

 * Add more unit tests
 * Optimize for efficiency
 * Add cloudformation custom resource script (will do this as soon as AWS golang support rolls out)

## Authors

* **David Gibb**

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details

