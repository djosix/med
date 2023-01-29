

TODO
- debug ctx.OutputPacket panic because channel closed
- debug socks proc performance
- debug socks proc client ctrl+c hanging
- implement procs
    - get /remote/path [/local/path] tar and gz mode
    - put /local/path [/remote/path] tar and gz mode
    - forward
        - forward local LOCAL_ENDPOINT REMOTE_ENDPOINT
            (listen on LOCAL_ENDPOINT -> connect to REMOTE_ENDPOINT)
        - forward remote REMOTE_ENDPOINT LOCAL_ENDPOINT
            (listen on REMOTE_ENDPOINT -> connect to LOCAL_ENDPOINT)
    - proxy
        - proxy socks5
        - proxy socks4a
        - proxy https
        - proxy http
    - webui ENDPOINT
    - tmux [TMUX_SOCK_PATH]
        - open a window when each med server connects back to med
    - prompt
        - med> get REMOTE_PATH [LOCAL_PATH]
        - med> put LOCAL_PATH [REMOTE_PATH]
        - med> forward local LOCAL_ENDPOINT [REMOTE_ENDPOINT]
        - med> proxy TYPE ENDPOINT [FLAGS]
        - med> self [kill] [remove]
- predefined build variants
    - separated: medc (client, keygen), meds (server)
    - combined: med (client, keygen, server)
- random keypair at build
    - generate keypairs using codegen?
- user-specified magic
- use protobuf instead of gob
- history: for pentesters
    - save actions (procs & actions)
    - save raw tcp streams to .pcapng files
- relay: connect two clients or two servers
    - while true; do cat -l 12333 -c 'ncat -l 12333'; sleep 1; done
- if dispatcher get error, close the proc