# Verifying Haven Binaries

This document provides instructions on how to verify the authenticity of Haven binary releases.

## Obtaining Release Files

The release binaries, along with `checksums.txt` and `checksums.txt.sig` files, can be downloaded from the official GitHub releases page:

[https://github.com/bitvora/haven/releases](https://github.com/bitvora/haven/releases)

Download the appropriate binary for your system along with the checksum files.

## Obtaining PGP Keys

Before verifying the binaries, you need to obtain the Haven PGP keys. You can do this in two ways:

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

If the verification is successful, you can be confident that the binary has not been tampered with and is the official 
release from the Haven team.

## Troubleshooting

If you encounter any issues during verification, please:

1. Ensure you've downloaded all required files
2. Check that you've imported the PGP keys correctly
3. Verify that you're using the correct commands for your operating system

---

### GPG trust and the "not certified with a trusted signature" warning

When verifying checksum signatures, you may see output like this:

```
$ gpg --with-fingerprint --verify checksums.txt.sig checksums.txt
gpg: Signature made Tue 12 Aug 12:03:12 2025 BST
gpg:                using EDDSA key 59F3BB93E8F4097CC43ED03C66865E933089774D
gpg: Good signature from "Haven <haven@bitvora.com>" [unknown]
gpg: WARNING: This key is not certified with a trusted signature!
gpg:          There is no indication that the signature belongs to the owner.
Primary key fingerprint: 1924 3581 B019 B245 2DA2  F828 70FF 8598 9022 1E23
     Subkey fingerprint: 59F3 BB93 E8F4 097C C43E  D03C 6686 5E93 3089 774D
```

#### What this means:

* **Good signature** confirms the file was signed with the private key corresponding to the listed public key.
* The warning about the key **not certified with a trusted signature** means your local GPG keyring has not established 
trust for this key via the PGP Web of Trust. It does **not** mean the file signature is invalid. It simply indicates 
that you (or your trusted introducers) have not attested ownership of this key.

* In short: the warning does not mean the file is invalid, only that you haven’t explicitly trusted the signer yet

#### Is it safe to proceed?

* For the purposes of verifying the Haven binary, a Good signature from Haven’s latest signing subkey (currently
`59F3 BB93 E8F4 097C C43E  D03C 6686 5E93 3089 774D`), with the primary fingerprint
`1924 3581 B019 B245 2DA2  F828 70FF 8598 9022 1E23`, is sufficient.
* Always ensure the fingerprints in the output match exactly what is shown above.

### Optional: sign and publish the key

Experienced PGP users may choose to add their trust attestation by signing Haven’s key and publishing it to
`keys.openpgp.org`. By doing so, you are attesting that you trust this key to belong to 
*Haven <haven@bitvora.com>*.

There is no "passport" for a software signing key. Trust is established socially and cryptographically through 
attestations.

#### Example commands:

1. Ensure you have Haven's key in your keyring:

    ```bash
    gpg --keyserver hkps://keys.openpgp.org --recv-keys 19243581B019B2452DA2F82870FF859890221E23
    ```

2. Sign Haven’s key with your own key:

    ```bash
    gpg --local-user YOUR_SIGNING_KEY_ID --sign-key 19243581B019B2452DA2F82870FF859890221E23
    ```

3. Optionally publish the signed key to keys.openpgp.org:

    ```bash
    gpg --keyserver hkps://keys.openpgp.org --send-keys 19243581B019B2452DA2F82870FF859890221E23
    ```

#### Notes

* `keys.openpgp.org` may require email verification for user IDs to be displayed. Even if UIDs are hidden until 
verified, uploading the key still publishes updated signatures.
* If this process feels too complex, you can also manually edit the key in your keyring:

  ```bash
  gpg --edit-key 19243581B019B2452DA2F82870FF859890221E23
  ```

  Then use the `trust` command to set trust level `5` (ultimate). This works around the warning, but it is 
**not recommended**, as ultimate trust should be reserved for your own keys.
