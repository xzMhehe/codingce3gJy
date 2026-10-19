#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""tgz_check.py - STRONG integrity check for a .tar.gz package.

Windows built-in tar.exe (bsdtar) is known to crash (0xC0000005) leaving a
TRUNCATED/CORRUPT gzip that md5 cannot detect. Python tarfile's lazy reading
also misses a broken gzip footer (it only iterates members). This checker
fully decompresses the gzip stream (CRC footer must validate) AND enumerates
every tar member, so a corrupt package is guaranteed to fail here.

Usage:  python tgz_check.py <file.tar.gz>
Exit:   0 = OK, 1 = corrupt
"""
import gzip
import sys
import tarfile


def main() -> int:
    if len(sys.argv) < 2:
        print('usage: python tgz_check.py <file.tar.gz>')
        return 2
    path = sys.argv[1]
    # 1) full gzip decompression -> validates the CRC32 footer
    with gzip.open(path, 'rb') as f:
        while f.read(1024 * 1024):
            pass
    # 2) enumerate every member (a truncated tar raises here)
    n = 0
    with tarfile.open(path, 'r:gz') as t:
        for _ in t:
            n += 1
    print('OK {0}: {1} members, gzip stream intact'.format(path, n))
    return 0


if __name__ == '__main__':
    try:
        sys.exit(main())
    except Exception as e:  # noqa: BLE001 - report any integrity failure
        print('CORRUPT {0}: {1}'.format(sys.argv[1], e))
        sys.exit(1)
