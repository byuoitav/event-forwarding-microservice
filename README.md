## event-forwarding-microservice
The event-forwarding-microservice receives events from the central event hub and forwards them to Redis cache and logging systems like ELK and Humio. Caches and logging systems are configured in a json file: service-config.json. 

### service-config.json Format

```
{
        "caches": [
                {
                        "name": "default",
                        "cache-type": "redis",
                        "type-cache": {
                                "device-index": "oit-static-av-devices-v3",
                                "room-index": "oit-static-av-rooms-v3",
                                "url": "location.byu.edu:port"
                        },
                        "generic-cache": {
                                "device-database": 0,
                                "room-database": 1,
                                "url": "wherever.amazonaws.com:port"
                        }
                }
        ],
        "forwarders": [
                {
                        "name": "LogEvents",
                        "type": "logging_system_name",
                        "event-type": "all",
                        "data-type": "event",
                        "cache-name": "default"
                        "logging_system_name": {
                                "update-interval": 5,
                                "buffer-size": 4000
                        }
                }
        ]
} 
```
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
## Cache Options
### redis-cache
```
"redis-cache": {
                                "device-database": 0,
                                "room-database": 1,
                                "url": "production.location.com:PORT"
                        }
```
