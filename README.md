# rbdl - RepeaterbookDL CLI

A command-line tool to download repeater data from the [RepeaterBook API](https://www.repeaterbook.com/).

## Important Usage Notice

**This application is for personal use only.** By using this tool, you agree to comply with [RepeaterBook's Terms of Service](https://www.repeaterbook.com/wiki/doku.php?id=api).

Key restrictions:
- Personal, non-commercial use only
- Do not redistribute the data without written consent from RepeaterBook
- Always credit RepeaterBook as the data source
- Commercial use requires authorization and potential fees

**Note:** In their API documentation, RepeaterBook indicates that "API tokens will be required for continued commercial usage starting **January 6, 2025**". However, at the time of writing (November 2025), only an application name and email address are required to authenticate with the API.

## Installation

### Prerequisites
- Go 1.21 or higher

### Build from source
```bash
git clone https://github.com/cartertemm/rbdl.git
cd rbdl
# On windows:
go build -o rbdl.exe
# On other platforms
go build -o rbdl
```

This will create an `rbdl` executable (or `rbdl.exe` on Windows) in the current directory.

### Install to PATH
```bash
go install
```

## Usage

### Basic Syntax
```bash
rbdl [options]
```

### Required Configuration
An email address is required for the API User-Agent header. You can provide it in two ways:

**Option 1: Command-line flag**
```bash
rbdl --email your.email@example.com [other options]
```

**Option 2: Environment variable**
```bash
export RBDL_EMAIL=your.email@example.com
rbdl [other options]
```

### Search Parameters

All search parameters are optional and can be combined:

| Flag | Description | Example |
|------|-------------|---------|
| `--email` | Email address (required) | `--email user@example.com` |
| `--output` | Output file path | `--output results.json` |
| `--callsign` | Repeater callsign | `--callsign W6ABC` |
| `--city` | Repeater city | `--city "San Francisco"` |
| `--country` | Repeater country | `--country USA` |
| `--frequency` | Repeater frequency | `--frequency 146.52` |
| `--mode` | Operating mode | `--mode DMR` |
| `--landmark` | Landmark | `--landmark "Golden Gate"` |
| `--state` | State/Province FIPS code | `--state CA` |
| `--region` | Region (international) | `--region Europe` |
| `--stype` | Service type | `--stype GMRS` |

### Wildcard Searches

Use `%` as a wildcard for pattern matching:

- **Prefix match:** `--callsign W%` (matches all callsigns starting with W)
- **Suffix match:** `--city %ville` (matches cities ending with "ville")
- **Contains:** `--callsign %ABC%` (matches callsigns containing "ABC")

### Examples

**Search by state and mode:**
```bash
rbdl --email user@example.com --state CA --mode DMR
```

**Search by frequency in a specific country:**
```bash
rbdl --email user@example.com --country USA --frequency 146.52
```

**Wildcard search for callsigns:**
```bash
rbdl --email user@example.com --callsign "K6%"
```

**Combine multiple parameters:**
```bash
rbdl --email user@example.com --state NY --mode P25 --output ny_p25.json
```

**Fetch all data (no search parameters):**
```bash
rbdl --email user@example.com
```

### Output

The application saves data as JSON to disk:

- **Specified output:** Use `--output` to specify a custom filename
- **Auto-generated:** If `--output` is not provided, a filename is generated automatically based on search parameters and timestamp

Example auto-generated filename:
```
repeaterbook_state_CA_mode_DMR_20250108_143022.json
```

## Operating Modes

The following operating modes are supported:
- `analog`
- `DMR`
- `NXDN`
- `P25`
- `tetra`

## Rate Limiting

The RepeaterBook API implements rate limiting. If you exceed the limits, you'll receive an error:
```
Error: rate limit exceeded (429): too many requests. Wait 10-60 seconds before retrying
```

Wait 10-60 seconds before making another request.

## Troubleshooting

### "email is required" error
Make sure you've set your email either via `--email` flag or the `RBDL_EMAIL` environment variable.

### Rate limit errors (429)
Wait at least 10-60 seconds before retrying your request. The API is designed for normal human interaction, not automated bulk downloads.

### Invalid JSON response
This usually indicates an API error. Check the error message for details.

## Contributing

Contributions are welcome! Please ensure all changes:
- Follow Go best practices
- Maintain compatibility with Linux, macOS, and Windows
- Include appropriate error handling

## License

This tool is provided as-is for personal use. The data accessed through this tool is owned by RepeaterBook and subject to their terms of service.

## Credits

Data provided by [RepeaterBook](https://www.repeaterbook.com/).
