# OG

OG  is dag based decentralized ledger and Dapp platform

## build
    go get github.comm/annchain/og
    make og

### run
    running og by loading config file from local filesystem :
    ./og -c  config.toml  run
    running og by loading config file from remote http server :
    ./og -c http://127.0.0.1:8532/og_config?node_id=3  run

###help commands
  ./og --help
   OG to da moon

  Usage:
    OG [command]

  Available Commands:
    help        Help about any command
    run         Start a full node

  Flags:
    -c, --config string         Path for configuration file or url of config server (default "config.toml")
    -d, --datadir string        Runtime directory for storage and configurations (default "data")
    -h, --help                  help for OG
    -l, --log_dir string        Path for configuration file. Not enabled by default
    -v, --log_level string      Logging verbosity, possible values:[panic, fatal, error, warn, info, debug] (default "debug")
    -n, --log_line_number       write log with line number
    -s, --log_stdout            Whether the log will be printed to stdout
    -m, --multifile_by_level    multifile log by level
    -M, --multifile_by_module   multifile log by module

## License

    This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
