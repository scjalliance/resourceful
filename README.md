# resourceful

[![GoDoc](https://godoc.org/github.com/scjalliance/resourceful?status.svg)](https://godoc.org/github.com/scjalliance/resourceful)

Resourceful provides lease-based management of finite resources. Policy files
describe the resources to be managed and the limits that should be imposed on
each resource. A resourceful server acts as a guardian that observes the
policies. Clients voluntarily connect to guardian servers to request leases
permitting resource consumption. If no active leases are available, clients may
receive a queued lease while they wait for an active lease to become available.
Active leases are assigned on a first come first serve basis.

The leases that clients receive are time-limited. Clients must renew them for
as long as they continue to consume or wait for a resource. If an active lease
expires without renewal the client must cease its use of that resource.

Resourceful was originally written to limit the number of concurrent uses of
pay-for-use software. Limiting the total number of concurrently running
instances of a particular program is a typical use case for resourceful.

The resourceful executable is capable of acting as either client or server. Its
role is determined by the arguments with which it is invoked. Clients are
invoked with `resourceful run` and servers are invoked with
`resourceful guardian`.

For windows clients, resourceful has a graphical user interface that will be
displayed when it fails to acquire an active lease. Windows clients are capable
of locating their resourceful server by looking up DNS service records located
in the Windows domain to which the computers are joined.

A Docker image of the guardian server is available on [Docker Hub](https://hub.docker.com/r/scjalliance/resourceful/).

## Example Docker Invocation

```
docker run -d --name=resourceful --restart=always -p 0.0.0.0:5877:5877 -e POLICY_PATH=/data/policies -e LEASE_STORE=bolt -e BOLT_PATH=/data/leases/leases.boltdb -e TRANSACTION_LOG=/data/tx/resourceful.tx.log -e CHECKPOINT_SCHEDULE=6h,200ops -v /data/resourceful/policies:/data/policies:ro -v /data/resourceful/leases:/data/leases -v /data/resourceful/tx:/data/tx scjalliance/resourceful
```

## Example DNS Service Record

```
_resourceful._tcp.contoso.com. 900 IN     SRV     0 100 5877 resourceful.contoso.com.
```

## Example Windows Shortcut

```
"C:\Program Files\SCJ\resourceful\resourceful.exe" run "%PROGRAMFILES(x86)%\Bentley\MicroStation V8i (SELECTseries)\MicroStation\ustation.exe" -wsWSDOT_Resources=%PROGRAMDATA%\WSDOT\CAE\1.3\
```

## Example Policy Files

```
{
        "resource": "notepad",
        "environment": {
                "resource.name": "Notepad"
        },
        "criteria": [{"component": "resource", "comparison": "regex", "value": ".*notepad(.exe)?$"}],
        "strategy": "consumer",
        "limit": 1,
        "duration": "2m",
        "decay": "10s",
        "refresh": {
                "active": "8s",
                "queued": "4s"
        }
}
```

```
{
        "resource": "bentley-openroads-designer",
        "environment": {
                "resource.name": "Power InRoads V8i"
        },
        "criteria": [{"component": "resource", "comparison": "regex", "value": ".*PowerInRoads(.exe)?$"}],
        "strategy": "consumer",
        "limit": 2,
        "duration": "15m",
        "decay": "15m",
        "refresh": {
                "active": "1m",
                "queued": "5s"
        }
}
```

```
{
        "resource": "bentley-microstation",
        "environment": {
                "resource.name": "MicroStation V8i"
        },
        "criteria": [{"component": "resource", "comparison": "regex", "value": ".*ustation(.exe)?$"}],
        "strategy": "consumer",
        "limit": 1,
        "duration": "15m",
        "decay": "15m",
        "refresh": {
                "active": "1m",
                "queued": "5s"
        }
}

```

## Server Environment Variables

```
LEASE_STORE
BOLT_PATH
POLICY_PATH
TRANSACTION_LOG
CHECKPOINT_SCHEDULE
```
