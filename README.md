# PDF plugin

This plugin has helper functions for working with PDF files.

## Installation

Follow the [instructions](https://docs.halon.io/manual/comp_install.html#installation) in our manual to add our package repository and then run the below command.

### Ubuntu

```
apt-get install halon-extras-pdf
```

### RHEL

```
yum install halon-extras-pdf
```

> [!IMPORTANT]
> When using RHEL you need to manually install `weasyprint` binary so it can be executed globally if you want to create PDF files from `text/html`.

### Azure Linux

```
tdnf install -y halon-extras-pdf
```

> [!IMPORTANT]
> When using Azure Linux you need to manually install `weasyprint` binary so it can be executed globally if you want to create PDF files from `text/html`.

## Exported classes

These classes needs to be [imported](https://docs.halon.io/hsl/structures.html#import) from the `extras://pdf` module path.

### PDF(data [, options])
Create a new PDF.

**Params**

- data `string` - the content to create the PDF from
- options `array` - an options array

The following options are available in the options array.

- format `string` - The format for the content. Can be `text/plain` or `text/html`. The default is `text/plain`.

**Returns**: class object

```
$x = PDF("hello world");
$x->addAttachment("test.txt", "hello");
$x->addAttachment("test2.txt", "world");
echo $x->toString(["password" => "12345678"]);
```

#### addAttachment(name, data [, options])
Add an attachment to the PDF. On error a exception is thrown.

**Params**

- name `string` - the name
- data `string` - the data
- options `array` - an options array

The following options are available in the options array.

- desc `string` - The description for the attachment.

**Returns**: this

**Return type**: `PDF`

#### toString([options])
Return the PDF file as a string. On error a exception is thrown.

**Params**

- options `array` - an options array

The following options are available in the options array.

- password `string` - If the PDF should be encrypted (AES256)

**Returns**: pdf data

**Return type**: `string`
