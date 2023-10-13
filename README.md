### Usage

```sh
cf-ddns --resolver https://public-ip.example.com --token <cloudflare api token> --zone <zone id> --records <record 1 id> --records <record 2 id>
```

Where the `-resolver` flag points to an endpoint that returns the public IP as a string.

Multiple `--records` flags can be specified to set more than one DNS record at a time.
