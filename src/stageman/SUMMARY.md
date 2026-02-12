# Stage Manager (stageman) - Complete Reorganization Summary

## What Changed

### 1. Renamed from workflow-lib to stageman

**Old name:** `workflow-lib.sh` (Workflow Library)
**New name:** `stageman.sh` (Stage Manager)

**Rationale:** "Stage Manager" better describes the utility's purpose as the canonical query interface for stage and state definitions.

### 2. Reversed Directory Structure

**Before (src-first):**
```
src/workflow-lib/
└── workflow-lib.sh          # Main implementation

fab/.kit/scripts/
└── workflow-lib.sh          # Symlink → src/workflow-lib/workflow-lib.sh
```

**After (kit-first):**
```
fab/.kit/scripts/
└── stageman.sh              # Main implementation (distribution)

src/stageman/
└── stageman.sh              # Symlink → ../../fab/.kit/scripts/stageman.sh
```

**Rationale:** Main file lives in the distributed kit, development symlink points to it.

## Final Structure

```
fab/.kit/
├── scripts/
│   └── stageman.sh          ← Main implementation
└── schemas/
    ├── workflow.yaml        ← Schema definition
    ├── README.md            ← Schema documentation
    └── MIGRATION.md         ← Migration guide

src/stageman/
├── stageman.sh              ← Symlink to main file
├── test-simple.sh           ← Basic smoke tests
├── test.sh                  ← Comprehensive suite (WIP)
├── README.md                ← Development guide
├── SPEC.md                  ← API specification
├── CHANGELOG.md             ← Version history
└── SUMMARY.md               ← This file
```

## Usage

### As Command

```bash
# From kit (distributed file)
fab/.kit/scripts/stageman.sh --help
fab/.kit/scripts/stageman.sh --version
fab/.kit/scripts/stageman.sh --test

# From src (via symlink)
src/stageman/stageman.sh --help
```

### As Library

```bash
# In bash scripts
source fab/.kit/scripts/stageman.sh

# Query functions
get_all_stages              # List all stage IDs
get_stage_number "spec"     # Get position (2)
get_state_symbol "active"   # Get symbol (●)
validate_status_file path   # Validate .status.yaml
```

### Development

```bash
# Edit main file
vim fab/.kit/scripts/stageman.sh

# Run tests (via symlink)
src/stageman/test-simple.sh

# Tests automatically use updated main file
```

## Benefits

1. **Clear distribution model** - Main file lives where it's used (.kit/scripts/)
2. **Simple development** - Edit one file, test via symlink
3. **Better naming** - "stageman" clearly indicates purpose
4. **Consistent structure** - Follows fab-kit patterns

## API (20+ Functions)

### State Queries
- `get_all_states` - List all valid states
- `validate_state <state>` - Check if state is valid
- `get_state_symbol <state>` - Get display symbol
- `is_terminal_state <state>` - Check if terminal

### Stage Queries
- `get_all_stages` - List all stages in order
- `validate_stage <stage>` - Check if stage exists
- `get_stage_number <stage>` - Get 1-indexed position
- `get_stage_name <stage>` - Get display name
- `get_stage_artifact <stage>` - Get generated filename
- `get_allowed_states <stage>` - List allowed states
- `get_initial_state <stage>` - Get default state
- `is_required_stage <stage>` - Check if required
- `has_auto_checklist <stage>` - Check if generates checklist

### Progression
- `get_current_stage <file>` - Detect active stage
- `get_next_stage <stage>` - Get next stage

### Validation
- `validate_status_file <file>` - Full validation
- `validate_stage_state <stage> <state>` - Check allowed

### Display
- `format_state <state>` - Format for display

## Testing

```bash
# Quick smoke test
src/stageman/test-simple.sh

# Self-test from main file
fab/.kit/scripts/stageman.sh --test

# Example output:
Testing stageman...

All states:
pending
active
done
skipped
failed

All stages:
brief
spec
tasks
apply
review
archive

✓ All tests passed
```

## Version

- **stageman:** 1.0.0
- **Schema:** 1.0.0
- **Compatible with:** fab-kit >= 0.1.0

## See Also

- [README.md](README.md) - Development guide
- [SPEC.md](SPEC.md) - Complete API documentation
- [CHANGELOG.md](CHANGELOG.md) - Version history
- [../../fab/.kit/schemas/workflow.yaml](../../fab/.kit/schemas/workflow.yaml) - Schema definition
- [../../fab/.kit/schemas/README.md](../../fab/.kit/schemas/README.md) - Schema documentation
