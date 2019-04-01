# Build

Command to build the CLI:

```sh
go build github.com/yinyin/go-literal-code-generator/cmd/go-literal-code-gen
```

# Example

````````markdown
# Heading Code

* `tail-new-line`

```
package literal

import (
	"strconv"
)

# Literal Constant 1

* `const`: `literalOne`
* `strip-spaces`

```
An apple a day
```

````````

# Options

* `const`: `(CONSTANT_NAME)` - Generate constant.
* `builder`: `(FUNCTION_NAME)`, `(PARAMETER_DEFINITIONS)` - Generate builder function.
* `strip-spaces` - Remove prefix and suffix spaces.
* `preserve-new-line` - Generate new line character for all lines.
* `tail-new-line` - Generate tail new line character.
