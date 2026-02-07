# Web of Trust

The WoT depth can be configured using the `WOT_DEPTH` environment variable:
- **Level 0**: Disabled. The relay is public, and anyone can write to the Inbox and Chat relays.
- **Level 1**: Private. Only the relay owner can write to the Inbox and Chat relays.
- **Level 2**: Following. Only the relay owner and the people they follow directly can write to the Inbox and Chat relays.
- **Level 3**: Connection of Connections. The relay owner, the people they follow, and the people followed by them can write to the Inbox and Chat relays. This is the default setting.

### Other Settings

- `WOT_MINIMUM_FOLLOWERS`: The minimum number of common followers required for someone to be included in your Web of Trust at Level 3.
- `WOT_FETCH_TIMEOUT_SECONDS`: The maximum time (in seconds) the relay will wait for a response when fetching Web of Trust data from other relays. Default is 30.
- `WOT_REFRESH_INTERVAL`: How often the relay should refresh its Web of Trust data. Supports duration strings like `24h`, `1h`, etc. Default is 24h.
