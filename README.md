## event-forwarding-microservice
### Production Branch is Main
 [![Apache 2 License](https://img.shields.io/hexpm/l/plug.svg)](https://raw.githubusercontent.com/byuoitav/touchpanel-ui-microservice/master/LICENSE)  
The event-forwarding-microservice receives events from the central event hub and forwards them to logging systems like ELK and Humio. Logging systems are configured in a json file: service-config.json. 

### service-config.json Format

```
{
        "forwarders": [
                {
                        "name": "LogEvents",
                        "type": "logging_system_name",
                        "event-type": "all",
                        "data-type": "event",
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
## Humio Parser Settings
This is the Parser Script for Humio that will correctly parse the received Json and accompanying timestamp
```
parseJson() | parseTimestamp("millis", field=@timestamp)
```

