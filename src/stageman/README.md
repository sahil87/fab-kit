# Stage Manager (stageman) - Development

Stage Manager is a bash utility for querying workflow stages and states from the canonical schema.

## Files

- **stageman.sh** - Symlink to main implementation at `fab/.kit/scripts/stageman.sh`
- **test-simple.sh** - Basic functionality tests
- **test.sh** - Comprehensive test suite (WIP)
- **README.md** - This file

## Directory Structure

```
fab/.kit/scripts/
└── stageman.sh          # Main implementation (distribution file)

src/stageman/
├── stageman.sh          # Symlink → ../../fab/.kit/scripts/stageman.sh
├── test-simple.sh       # Basic smoke tests
├── test.sh              # Full test suite
├── README.md            # Development documentation
├── SPEC.md              # API specification
└── CHANGELOG.md         # Version history

fab/.kit/schemas/
└── workflow.yaml        # Schema definition (queried by stageman)
```

## Quick Start

```bash
# Test the utility
fab/.kit/scripts/stageman.sh --version
src/stageman/test-simple.sh

# Use in scripts
source fab/.kit/scripts/stageman.sh
get_all_stages
get_stage_number "spec"
```

## See Also

- [SPEC.md](SPEC.md) - Complete API documentation
- [CHANGELOG.md](CHANGELOG.md) - Version history
- [../../fab/.kit/schemas/README.md](../../fab/.kit/schemas/README.md) - Schema documentation
