# ph

`ph` was build to limit the game time of my kids.

It is a tool that monitors OS processes and terminates those that exceed the specified *time limit* for the day, and prevents processes to run during the specified *downtime*

## Configuration

Time limits and downtime periods are specified in a configuration file `cfg.json`, in `JSON` format, like this:

```json
[
    {
        "processes": [
            "example process 1",
            "example process 2.exe"
        ],
        "limits": {
            "*": "1h",
            "mon wed fri 2019-12-25": "2h30m"
        },
        "downtime": {
            "*": [
                "..12:00",
                "13:30..14:00",
                "22:00.."
            ]
        }
    },
    {
        "processes": [
            "FortniteClient-Win64-Shipping.exe",
            "RustClient.exe"
        ],
        "limits": {
            "*": "2h",
            "fri sat sun": "3h30m"
        }
    }
]
```

When the time balance of a process for the day exceeds the specified time limit for that day (defined in `limits` section), the process is terminated. The process is also terminated during the specified downtime (defined in `downtime` section).

When more than one process names are specified in `processes` group (as array or strings), then all these processes will contribute to the group's time balance for the day. Processes belonging to a groups will be terminated if the time balance of the group exceeds the specified limit (if defined), or during downtime periods (if defined)

Time limits are in the `"HHhMMhSSs"` format, where `HH` is hours, `MM` - minutes and `SS` seconds. For example `3h45m30s` is a time limit of 3 hours, 45 minutes and 30 seconds for a particular day.

Downtime periods are in the `"HH:MM..HH:MM"` format (where downtime is between the two hours of the day), where time is specified in 24 hours format. `"..HH:MM"` and `"HH:MM.."` are also valid downtime periods.

Time limits `"limits"` and downtime periods `"downtime"` can be assigned to:

+ any day `"*"`
+ one or more specific days of the week or concrete dates, for example:
  + `"tue"` - for Tuesdays
  + `"2019-10-16"` - for Oct 16th, 2019
  + `"sat sun 2019-12-25"` - for Sundays, Saturdays or specifically for Dec 25th 2019

If a particular day matches more than one spec, then the most-concrete spec will be applied, in the following priority order:

+ concrete date, e.g. `"2019-10-16"`
+ concrete date from a `list` of days/dates, e.g. `"sat sun 2019-12-25"`
+ concrete day of week, e.g. `"mon"`
+ concrete day of week from a `list`, e.g. `"mon tue"`
+ any day, i.e. `"*"`

The days of the week are specified in format of three-letter abbreviations - `mon tue wed thu fri sat sun`.
Dates are in format `yyyy-mm-dd`.
When in `list`, days of the week or dates are separated by spaces.

### Time balance check

`ph` checks running processes once every three minutes (hardcoded).

### Configuration update

`ph` monitors for changes in the configuration file (`cfg.json`) and reloads it, if changes are detected. To change the configuration, just overwrite the configuration file.

The configuration can also be changed through the web UI and through the API at the [/config] endpoint.

## UI

The tool serves a simple, yet usable, web UI at [localhost:8080](localhost:8080).

Configuration can be edited through the web UI, but requires authentication with username and password. Credentials are hard-coded in `server\server.go`.

## OS compatibility

`ph` is a multi-platform tool that runs on Linux, macOS and Windows.

### Windows

On Windows, `ph` is designed to work as a Windows service.

To build, install and run the tool as a Windows service, run `make build`, copy the `\bin` folder somewhere and run `phsvc install` and `phsvc start`.  

To enable the service to start automatically when Windows starts, open the Windows Service Manager, right click on `Process Hunter` service, select `Properties` from the context menu and set `Startup type` to `Automatic`.

Run `phsvc stop` and `phsvc remove` to stop and uninstall the service.

Run `phsvc debug` to run the `ph` as a command line tool (without installing as Windows service).

## Work in Progress

The tool is usable as it is, but can be improved. The author intends to develop it further, time permitting.

Top priority items are:

+ add tests for web UI (JavaScript scripts)
+ create installation scripts
+ make server port and credentials configurable
