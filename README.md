# Requirements

1. Go v1.23.6 or above: [download link](https://go.dev/doc/install).
2. Tinygo v0.36.0 or above: [installation instructions](https://tinygo.org/getting-started/install/).
3. Code editor, ex. VS Code.
4. GNU Make.
5. Git.
6. Raspberry PI Pico W

# Configurations

Before building and flashing the project, create a `config.json` file under `./internal/runner`. Configurable variables:

```json
{
    "SSID": "Wifi name. Only 2.4Ghz wifi connection should be used",
    "Password": "Wifi password",
    "IP": "Static IP, ex.: 192.168.100.100",
    "Port": numeric port, ex.: 1234,
    "Hostname": "any name for the host"
}
```

# Troubleshooting

1. Can't monitor Pico board using USB connection on Linux?

Try running this command to add current user to `dialout` group:
> sudo usermod -aG dialout $USER

If you're using Immutable linux and the above command does not add user to the group, you will need to do it manually:
- Go to `/etc/` and find `group` file, open it as Administrator (using `sudo`),
- Add additional line `dialout:x:18:{current user name}`, user name can be found using:
> whoami