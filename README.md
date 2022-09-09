# Adafruit Bot

Auto-checkout tool to purchase products on Adafruit.com

## Configiration
Edit profile.json accordingly


Install go dependecies

```bash
go install
```
Build project
```bash
go build .
```

## Usage

```bash
./adafruit
[2022-09-08@21:06:32] Monitoring product...
[2022-09-08@21:06:33] Fetching product page
[2022-09-08@21:06:33] Attempting ATC
[2022-09-08@21:06:33] Added item to cart
[2022-09-08@21:06:33] Scraping CSRF token
[2022-09-08@21:06:34] Submitting billing details
[2022-09-08@21:06:45] Finalizing Order
[2022-09-08@21:06:48] Check email
```

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License
[GPL](https://choosealicense.com/licenses/gpl/)
