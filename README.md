# labelq

Carriers don't bill packages by their actual weight. They bill by the
*billable* weight: the greater of actual weight and dimensional weight,
where dimensional weight is a function of the box's volume. A big, light
box (a lampshade, a stack of pillows) can cost far more to ship than its
bathroom-scale weight suggests, and it's easy to miss that until the
invoice shows up.

labelq reads a manifest of shipments and reports, for each one, the
actual weight, the dimensional weight, and which one the carrier will
actually charge for. That's the one question it answers.

## Usage

```
$ labelq examples/sample.manifest
ID         ACTUAL       DIM   BILLABLE  NOTE
SH1001        13lb        4lb       13lb
SH1002         2kg        5kg        5kg  dimensional weight applies
SH1003         1lb       58lb       58lb  dimensional weight applies
```

SH1003 is a 20x20x20 inch box that weighs one pound. On the scale it's
nothing; on the invoice it's billed as 58 pounds.

## Manifest format

A manifest is plain text, made of `shipment ... end` blocks:

```
shipment SH1001
  weight 12.5 lb
  dims 10x8x6 in
  service ground
end
```

Rules:

- `weight <number> <lb|kg>` — actual scale weight.
- `dims <LxWxH> <in|cm>` — outer dimensions, e.g. `10x8x6`.
- `service <name>` — free-form service label (`ground`, `express`, ...).
- Weight and dims must use the same unit system: `lb` goes with `in`,
  `kg` goes with `cm`. labelq will not silently convert between them.
- Indentation is cosmetic; the parser only cares about tokens per line.
- Blank lines between blocks are allowed and ignored.

## Error messages

Manifests get hand-edited, and the usual failure mode is a typo in a
unit or a stray character in a number. labelq points at exactly where
the problem is instead of a bare "parse failed":

```
$ labelq broken.manifest
broken.manifest:2:15: invalid unit "lbs" (expected lb or kg)
    weight 12.5 lbs
                  ^
```

If a manifest has several problems, labelq reports all of them in one
pass, not just the first one it trips over.

## Dimensional weight formula

- Imperial: `(L * W * H in inches) / 139`, result in pounds.
- Metric: `(L * W * H in cm) / 5000`, result in kilograms.

These are the standard domestic divisors used by major carriers absent
a specific contract; both actual and dimensional weight are rounded up
to the nearest whole unit before comparing, which is how carriers bill.

## Build

```
go build ./...
```

No third-party dependencies; standard library only.

## License

MIT, see [LICENSE](LICENSE).
