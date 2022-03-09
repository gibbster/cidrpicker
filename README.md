# CIDR Picker

Checks to see if there's enough space in a VPC for a subnet of a given size. If there is, return it in the form of a CIDR block.

## Install

`go install github.com/gibbster/cidrpicker@master`

## Contribute

Clone the repository
`git clone git@github.com:gibbster/cidrpicker.git`

Build and test
`go build`
`go test`

## Run

Load autocompletion:

`source sourceme.bash`

Get help:

```
$ ./cidrpicker
NAME:
    cidrpicker - Return the next available CIDR block of the given size for a given VPC.

SYNOPSIS:
    cidrpicker [--help|-?] [--quiet] <command> [<args>]

COMMANDS:
    free    Find the next free CIDR block for the given size.
    list    List VPCs

OPTIONS:
    --help|-?    (default: false)

    --quiet      (default: false)

Use 'cidrpicker help <command>' for extra details.
```

```
$ ./cidrpicker free help  
NAME:
    cidrpicker free - Find the next free CIDR block for the given size.

SYNOPSIS:
    cidrpicker free --vpc-id <string> [--help|-?] [--number <int>]
                    [--profile <aws_profile>] [--quiet] [--region <aws_region>]
                    [--size <int>] [<args>]

REQUIRED PARAMETERS:
    --vpc-id <string>

OPTIONS:
    --help|-?                  (default: false)

    --number <int>             number of CIDR blocks to generate. (default: 1)

    --profile <aws_profile>    (default: "")

    --quiet                    (default: false)

    --region <aws_region>      (default: "")

    --size <int>               (default: 24)
```

Run the program:

```
$./cidrpicker list --profile <profile> --region <region>
VPC: vpc-XXX, CIDR: 10.0.0.0/16
```

```
$./cidrpicker free --profile <profile> --region <region> --vpc-id vpc-XXX -size 24 -n 2
10.0.2.0/24
10.0.3.0/24
```

## TODO

 * Add more unit tests
 * Optimize for efficiency
 * Add cloudformation custom resource script (will do this as soon as AWS golang support rolls out)
 * Once I learn more about golang standards - do that. This is my first attempt at Go, so I'm sure there's ugly warts to remove

## Author

* **David Gibb**

## Contributors

* **David Gamba**
