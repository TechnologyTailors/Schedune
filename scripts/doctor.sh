#!/usr/bin/env bash
set -e

echo "============================================"
echo "    Schedune Preflight & Doctor Check       "
echo "============================================"

PASS="[\033[32mOK\033[0m]"
WARN="[\033[33mWARN\033[0m]"
FAIL="[\033[31mFAIL\033[0m]"

# 1. OS Check
if [ "$(uname)" = "Linux" ]; then
    echo -e "$PASS Linux host detected: $(uname -m)"
else
    echo -e "$FAIL Schedune requires Linux. Found: $(uname)"
    exit 1
fi

# 2. SQLite / Persistence Check
DB_DIR="./var"
mkdir -p "$DB_DIR"
if [ -w "$DB_DIR" ]; then
    echo -e "$PASS Database directory writable: $DB_DIR"
else
    echo -e "$FAIL Database directory not writable: $DB_DIR"
fi

# 3. KVM Check
if [ -c "/dev/kvm" ]; then
    if [ -w "/dev/kvm" ]; then
        echo -e "$PASS /dev/kvm is present and writable"
    else
        echo -e "$WARN /dev/kvm exists but is not writable by current user. VM launch will fail."
    fi
else
    echo -e "$WARN /dev/kvm missing: VM execution is not supported on this host."
fi

# 4. QEMU Binary Check
QEMU_BIN="qemu-system-$(uname -m)"
if command -v "$QEMU_BIN" >/dev/null 2>&1; then
    echo -e "$PASS QEMU binary found: $(command -v $QEMU_BIN)"
else
    echo -e "$WARN $QEMU_BIN not found in PATH. KVM_QEMU backend will fail."
fi

# 5. Cloud Hypervisor Binary Check
if command -v "cloud-hypervisor" >/dev/null 2>&1; then
    echo -e "$PASS Cloud Hypervisor binary found: $(command -v cloud-hypervisor)"
else
    echo -e "$WARN cloud-hypervisor not found in PATH. CLOUD_HYPERVISOR backend will fail."
fi

# 6. Firecracker Checks
if command -v "firecracker" >/dev/null 2>&1; then
    echo -e "$PASS Firecracker binary found: $(command -v firecracker)"
else
    echo -e "$WARN firecracker not found in PATH. MicroVM launch validation will fail."
fi

if [ -c "/dev/net/tun" ]; then
    echo -e "$PASS /dev/net/tun is present (required for Firecracker networking)"
else
    echo -e "$WARN /dev/net/tun missing: Firecracker networking checks may fail."
fi

# 7. ProcFS
if [ -r "/proc" ]; then
    echo -e "$PASS /proc is readable (required for Orphan Sweeping)"
else
    echo -e "$FAIL /proc is not readable. Orphan detection will fail."
fi

echo ""
echo "============================================"
echo "    Preflight Complete                      "
echo "============================================"
