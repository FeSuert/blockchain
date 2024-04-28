from cryptography.hazmat.primitives.asymmetric import ed25519
from cryptography.hazmat.primitives import serialization

# Your private key (hexadecimal format)
private_key_hex = "d66f826786cdcde02cc2408b6c9a3aa1d9130a244b83a883d3aa3695fe417681"

# Convert the hex string to bytes
private_key_bytes = bytes.fromhex(private_key_hex)

# Create a private key object from the bytes
private_key = ed25519.Ed25519PrivateKey.from_private_bytes(private_key_bytes)

# Generate the public key
public_key = private_key.public_key()

# Serialize the public key to bytes and then convert to hex for display or use
public_key_bytes = public_key.public_bytes(
    encoding=serialization.Encoding.Raw,
    format=serialization.PublicFormat.Raw
)

public_key_hex = public_key_bytes.hex()

print("Public Key (hex):", public_key_hex)
