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
            "mon wed fri": "30m"
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

When more than one process name is specifed in `processes` group (as array or strings), then all these processes will contribute to the group's daily time balance. Processes belonging to such groups will be terminated if the time balance of the group exceeds the specified limit.

Daily time limits are pecified in the `"HHhMMhSSs"` format, where `HH` is hours, `MM` - minutes and `SS` seconds. For example `3h45m30s` specifies daily time limit of 3 hours, 45 minutes and 30 seconds.

Daily time limits `"limits"` are can be assigned to any day `"*"`, or to one or more specific days of the week (e.g. `"sat sun"` - for any of these two days, or just `"tue"` - for Tuesday). If a particular day matches more than one list, then the most-concrete specification will be applied.

Days are specified by space-separated list of lowercase, three-letter abbreviations, of the week days - `mon tue wed thu fri sat sun`

## UI

The tool serves a trivial web UI at [localhost:8080](localhost:8080)

## OS compatibility

`ph` is a multi-platform tool that runs on linux, macOS and Windows.

## Work in Progress

The tool is usable as it is, but far from perfect. The author intends to develop it further, mostly by imporving the user experience.

Top priority items are:

* reorganize the project ot use go modules
* make it Windows Service
* improve web UI
