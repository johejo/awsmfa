# awsmfa

A simple tool to update session token for aws cli mfa authentication.

## Simple Usage


Set the MFA device ID to your aws credential file.
```
[default]
aws_access_key_id     = YOUR_ACCESS_KEY_ID
aws_secret_access_key = YOUR_SECRET_ACCESS_KEY
aws_mfa_device        = YOUR_DEVICE_ID
```

Run command
```sh
awsmfa
```
```sh
aws s3 ls --profile mfa
// success
```

## License

MIT

## Author

Mitsuo Heijo
