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
