# sensu-influx-handler

A listener to take sensu metrics events on a TCP socket, and post them to InfluxDB

## Motivation

If you want to use Sensu as your monitoring router, you still need to send the metrics somewhere.  
InfluxDB is a good choice for this, but if you have a large number of metrics, forking a handler 
(using the `pipe` type) will be non-performant.

This small application is meant to run as a daemon, listening on a TCP port, for metrics events
from Sensu.

## Installation

You can use the [pre-built binaries](https://github.com/launchdarkly/sensu-influx-handler/releases), or follow the instructions in the [Compilation](#compilation) section to 
build yourself.

This process should be managed by something like [supervisord](http://supervisord.org).

Be sure to create an appropriate config file for your environment.  The environment is specifed using the `SENSU_INFLUX_MODE` environment variable.  For instance:

    SENSU_INFLUX_MODE=production ./sensu-influx-handler 

This will start the app listening on the port specified in the config file.

## Configuration

The configuration file used depends on the environment specified:

|`SENSU_INFLUX_MODE` value|config file used|
|---|---|
|(none)|./sensu-influx.local.conf|
|staging|./sensu-influx.stg.conf|
|production|./sensu-influx.prod.conf|

The configuration file should be in the following format:

    [influxdb]
    host = localhost:8086
    username = admin
    password = admin
    database = test
    isSecure = no
    isUDP = no

Then, configure a handler in Sensu, such as this `/etc/sensu/conf.d/handler_influxdb.json`:

    {
	  "handlers": {
	    "influxdb": {
    	  "type": "tcp",
	      "socket": {
    	    "host": "localhost",
        	"port": 3333
	      }
	    }
	  }
	}

## Compilation

*  To build it for your local system:

	   	godep go build

* To cross-build:

		goxc
