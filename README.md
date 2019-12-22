# ph

`ph` was build to limit the daily game time of my kids.

It is a tool that monitors OS processes and terminates those who exceed the specified daily time limit.

## Configuration

The daily time limit is specified in a configuration file `cfg.json`, like this:

```json
[
    {
        "processes": [
            "example process 1",
            "example process 2.exe",
        ],
        "limits": {
            "*": "1h",
            "mon wed fri 2019-12-25": "2h30m"
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

When daily time balance of a process exceeds the specified time limit, the process is terminated.

When more than one process names are specifed in `processes` group (as array or strings), then all these processes will contribute to the group's daily time balance. Processes belonging to such groups will be terminated if the time balance of the group exceeds the specified limit.

Daily time limits are in the `"HHhMMhSSs"` format, where `HH` is hours, `MM` - minutes and `SS` seconds. For example `3h45m30s` is a daily time limit of 3 hours, 45 minutes and 30 seconds.

Daily time limits `"limits"` can be assigned to:

+ any day `"*"`
+ one or more specific days of the week or concrete dates, for example:
  + `"tue"` - for Tuesdays
  + `"2019-10-16"` - for Oct 16th, 2019
  + `"sat sun 2019-12-25"` - for Sundays, Saturdays or specifically for Dec 25th 2019

If a particular day matches more than one specification, then the most-concrete specification will be applied, in the following priority order:

+ concrete date, e.g. `"2019-10-16"`
+ concrete date from a `list`, e.g. `"sat sun 2019-12-25"`
+ concreate day of week, e.g. `"mon"`
+ concreate day of week from a `list`, e.g. `"mon tue"`
+ any day, i.e. `"*"`

Week days are in format of three-letter abbreviations of the week days - `mon tue wed thu fri sat sun`.
Dates are in format `yyyy-mm-dd`.
When in `list` week days or dates are separated by spaces

## UI

The tool serves a trivial web UI at [localhost:8080](localhost:8080)

## OS compatibility

`ph` is a multi-platform tool that runs on Linux, macOS and Windows.

On Windows, `ph` is designed to work as Windows service. 

> To install as service, run `make build`, copy the `\bin` folder somewhere and run `phsvc install` and `phsvc start` to install and run the tool as Windows serive. Run `phsvc stop` and `phsvc remove` to stop and uninstall the service. Run `phsvc debug` to run the `ph` as a command line tool (without installing as Windows service)

## Work in Progress

The tool is usable as it is, but far from perfect. The author intends to develop it further, mostly by imporving the user experience.

Top priority items are:

- reorganize the project ot use go modules
- improve web UI
