## event-forwarding-microservice
### Production Branch is Main
 [![Apache 2 License](https://img.shields.io/hexpm/l/plug.svg)](https://raw.githubusercontent.com/byuoitav/touchpanel-ui-microservice/master/LICENSE)  
The event-forwarding-microservice receives events from the central event hub and forwards them to logging systems like ELK and Humio. Logging systems are configured in a json file: service-config.json. 

### service-config.json Format

```
{
    "caches": [
        {
            "name": "default",
            "cache-type": "memory"
        }
    ],
    "forwarders": [
        {
            "name": "ElkDeltaEvents",
            "type": "elktimeseries",
            "event-type": "delta",
            "interval": 10,
            "data-type": "event",
            "cache-name": "default",
            "elk": {
                "url": "http://localhost:9200/",
                "index-pattern": "av-delta-events",
                "index-rotation-interval": "monthly"
            }
        },
        {
            "name": "ElkAllEvents",
            "type": "elktimeseries",
            "event-type": "all",
            "interval": 10,
            "data-type": "event",
            "cache-name": "default",
            "elk": {
                "url": "http://localhost:9200/",
                "index-pattern": "av-all-events",
                "index-rotation-interval": "daily"
            }
        },
{
            "name": "ElkDeltaEvents",
            "type": "elktimeseries",
            "event-type": "delta",
            "interval": 10,
            "data-type": "event",
            "cache-name": "default",
            "elk": {
                "url": "http://localhost:9200/",
                "index-pattern": "av-delta-events",
                "index-rotation-interval": "monthly"
            }
        },
        {
            "name": "ElkAllEvents",
            "type": "elkstatic",
            "event-type": "all",
            "interval": 10,
            "data-type": "device",
            "cache-name": "default",
            "elk": {
                "url": "http://localhost:9200/",
                "index-pattern": "oit-static-av-devices-v3",
                "index-rotation-interval": "monthly"
            }
        }
    ]
}
```

## Endpoints
### Status
* <mark>GET</mark> `/ping` - Check if the microservice is running
* <mark>GET</mark> `/status` - Returns good if microservice is running

### Logging
* <mark>Get</mark> `/logLevel` - Get the current log level
* <mark>Get</mark> `/logLevel/:level` - Set the log level to the specified level
    * `debug`, `info`, `warn`, `error`


## Forwarder Options
### Humio
```
"humio": {
        "update-interval": 5, //send buffer every x seconds
        "buffer-size": 4000, //max amount of events that can be stored in a buffer
        "ingest-token": "jai52gwjl-auemdio5-5263-83lp-sjrd3853k9"
}
```
### Elk
```
"elk": {
        "url": "http://location.byu.edu:1534",
        "index-pattern": "av-delta-events", 
        "index-rotation-interval": "monthly"
}
```
## Humio Parser Settings
This is the Parser Script for Humio that will correctly parse the received Json and accompanying timestamp
```
parseJson() | parseTimestamp("millis", field=@timestamp)
```

