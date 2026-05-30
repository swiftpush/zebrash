# Zebrash

**Zebrash** is a library to convert **ZPL** (Zebra Programming Language) into: **PNG**, **PDF**, or **SVG**

## New maintainer
> This is a fork of the [ingridhq/zebrash](https://github.com/ingridhq/zebrash) library

## Description

This library emulates subset of ZPL engine and allows you to view most of the ZPL labels that are used by carriers such as Fedex, UPS or DHL as PNGs without the need to possess physical Zebra-compatible printer.
Think of https://labelary.com/viewer.html except it is completely free for commercial use, has no API limits and can easily be self-hosted or plugged into existing Go application so you don't need to send labels with real customers information to some 3rd-party servers

Example of the output (more examples can be found inside `testdata` folder):

![UPS label](testdata/ups_grayscale.png)

## Self-hosted, free, and private

Zebrash runs entirely inside your own application or infrastructure — there is no API to call and nothing to sign up for.

- **It's free.** No per-call quotas, no API keys, no subscription tiers. The library is MIT-licensed and free for commercial use.
- **Your data never leaves your machine.** Shipping labels carry real customer names, addresses, and tracking numbers. Because parsing and rendering happen locally, none of that is ever transmitted to a third party.
- **No external dependencies at runtime.** Zebrash works fully offline, with no outbound network calls, meaning no dependencies on third-party services

## Usage:

```go
exampleZPL := "^XA^FO50,50^FDHello World^FS^XZ"

drawer := zebrash.NewDrawer()

var buff bytes.Buffer
err = drawer.DrawLabelAsPng(exampleZPL, &buff, drawers.DrawerOptions{
	LabelWidthMm:         101.6,
	LabelHeightMm:        203.2,
	Dpmm:                 8,
	EnableInvertedLabels: true,
	GrayscaleOutput:      true
})
if err != nil {
	t.Fatal(err)
}

err = os.WriteFile("./testdata/label.png", buff.Bytes(), 0744)
if err != nil {
	t.Fatal(err)
}
```

## Contributing

Contributions are welcome! Please submit an issue or pull request.
For larger changes, please open an issue first to discuss the approach.

## License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.
