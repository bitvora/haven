# Verifying Haven Binaries

This document provides instructions on how to verify the authenticity of Haven binary releases.

## Obtaining Release Files

The release binaries, along with `checksums.txt` and `checksums.txt.sig` files, can be downloaded from the official GitHub releases page:

[https://github.com/bitvora/haven/releases](https://github.com/bitvora/haven/releases)

Download the appropriate binary for your system along with the checksum files.

## Obtaining GPG Keys

Before verifying the binaries, you need to obtain the Haven GPG keys. You can do this in two ways:

1. From the OpenPGP keyserver:

```bash
gpg --keyserver hkps://keys.openpgp.org --recv-keys 19243581B019B2452DA2F82870FF859890221E23
```

2. Directly from the Haven repository:

```bash
curl https://raw.githubusercontent.com/bitvora/haven/master/haven.asc -sSL | gpg --import -
```

## Verifying the Checksums File

First, verify that the `checksums.txt` file is authentic by checking its signature:

```bash
gpg --with-fingerprint --verify checksums.txt.sig checksums.txt
```

You should see output similar to:

```
gpg: Signature made Thu 17 Jul 01:14:35 2025 BST
gpg:                using EDDSA key 59F3BB93E8F4097CC43ED03C66865E933089774D
gpg: Good signature from "Haven <haven@bitvora.com>" [full]
Primary key fingerprint: 1924 3581 B019 B245 2DA2  F828 70FF 8598 9022 1E23
     Subkey fingerprint: 59F3 BB93 E8F4 097C C43E  D03C 6686 5E93 3089 774D
```

Make sure the fingerprints match the expected values and that you see "Good signature".

## Verifying the Binary Files

After confirming the checksums file is authentic, verify the binary you downloaded:

```bash
sha256sum -c --ignore-missing checksums.txt
```

For example, if you downloaded the macOS ARM64 binary, you should see:

```
haven_Darwin_arm64.tar.gz: OK
```

If the verification is successful, you can be confident that the binary has not been tampered with and is the official release from the Haven team.

## Troubleshooting

If you encounter any issues during verification, please:

1. Ensure you've downloaded all required files
2. Check that you've imported the GPG keys correctly
3. Verify that you're using the correct commands for your operating system