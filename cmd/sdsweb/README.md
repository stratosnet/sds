# sdsweb
`sdsweb` is a tool for running an SDS node with a management UI at the same time. It runs the same code as `ppd start`, while also exposing the [node monitor UI](https://github.com/stratosnet/stratos-node-monitor) on the configured port in the ppd config file (`web_server.port`)

### How to Build
Building the [SDS](https://github.com/stratosnet/sds) project will generate the binary for `ppd`, `relayd` and `sdsweb` at the same time.

    make install

### How to Run
Start by following the regular instructions for running an [SDS node](https://github.com/stratosnet/sds) (documentation [here](https://docs.thestratos.org/)).

Then instead of `ppd start`, run the following

    sdsweb start

The management UI should now be available by going to [http://localhost:18681](http://localhost:18681) in your local browser. Make sure to replace `18681` if you change `web_server.port` in the config file.