# OG

OG is a dag based decentralized ledger and Dapp platform.

## Insatallation
Get code from github:

```
go get github.com/annchain/OG
```

Build the project:

```
cd $GOPATH/src/github.com/annchain/OG

make og
```

The node will be built in `build` directory

## Running

```
# running og by loading config file from local filesystem :
./og -c config.toml  run

# running og by loading config file from remote http server :
./og -c http://127.0.0.1:8532/og_config run

# running og by assigning log output path:
./og -c config.toml -l log_path/ run
```

Other og commands:
```
-c, --config string         Path for configuration file or url of config server (default "config.toml")
-d, --datadir string        Runtime directory for storage and configurations (default "data")
-h, --help                  help for OG
-l, --log_dir string        Path for configuration file. Not enabled by default
-v, --log_level string      Logging verbosity, possible values:[panic, fatal, error, warn, info, debug] (default "debug")
-n, --log_line_number       write log with line number
-s, --log_stdout            Whether the log will be printed to stdout
-m, --multifile_by_level    split log into multiple files by level
-M, --multifile_by_module   split log into multiple files by module
```

## Usage

### RPC

Users can call the node's functions through RPC interface. The default URL address for RPC is `localhost:8000`, which is set in the default config file. You can switch the port by changing the value of key `[rpc].port` in config file.

For more details of each RPC API, you can check document: [https://github.com/annchain/OG/blob/dev/rpc/doc.md](https://github.com/annchain/OG/blob/dev/rpc/doc.md)

## Development

Please be aware that OG is under heavy refactor(under layer-p2p).

## License


<!-- 
This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details -->
