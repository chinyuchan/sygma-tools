# sygma-tools


## build
```
make build
```
## Generate local party
```
Generate ECDSA local party

Usage:
  sygma-tools ecdsa [flags]

Flags:
  -h, --help               help for ecdsa
  -p, --participants int   participants (default 2)
  -t, --threshold int      threshold (default 1)
```
* `p`: num of participants
* `t`: num of threshold

## Run
```
./sygma-relayer ecdsa -p 3 -t 1
```
## Result
The result json file in `test/_ecdsa_fixtures`.



