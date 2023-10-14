
A simple Go CLI tool to automatically set the content of A and AAAA records to a public IP for domains using Cloudflare DNS.

- set A and AAAA records automatically
- set multiple records within the same zone
- use any public IP resolver that returns the IP as plain text

### Usage

#### From the terminal as a command
```sh
cf-ddns --resolver https://public-ip.example.com --token <cloudflare api token> --zone <zone id> --records <record 1 id> --records <record 2 id>
```

Where the `-resolver` flag points to an endpoint that returns the public IP as a string.

Multiple `--records` flags can be specified to set more than one DNS record at a time.

#### As a service
Specify the environment variables and run the binary.

`-loop` will loop the service and continuously update the records until terminated.

`-period 5m` will specify the loop delay between periods (60s, 5m, 1h).

#### systemd 
Run the tool as a systemd service.
1. Build the binary and copy to the system with `make build && cp cf-ddns /bin/cf-ddns`
1. Copy [cf-ddns.service](./cf-ddns.service) file to `/etc/systemd/service/cf-ddns.service` and specify the environment variables.
2. Reload systemd configuration with `systemctl daemon-reload`
3. Enable and start the service with `systemctl enable --now`
4. View service logs with `journalctl -f -u cf-ddns`


### Environment Variables
`RESOLVER` The public IP resolver endpoint

`CLOUDFLARE_API_TOKEN` Your Cloudflare API token

`ZONE_ID` The ID of the Cloudflare Zone for the records you want to update

`RECORDS` A comma separate list of Cloudflare Records to update for the specified zone

### IP Resolver

An IP resolver must respond to the cf-ddns client with just the public IP as the body of the response.

#### Example
```
GET /ip HTTP/2
```

```
HTTP/2 200 OK

104.16.133.229
```

#### TODO
- [ ] use a nicer CLI library?
- [ ] optional structured logs with zerolog
- [ ] build a debian package to do the systemd install automatically