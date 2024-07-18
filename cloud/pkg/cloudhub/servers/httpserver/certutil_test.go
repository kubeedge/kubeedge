package httpserver

import (
	"encoding/pem"
	"testing"
)

func TestParseX509PrivateKey(t *testing.T) {
	ecKey := []byte(`
-----BEGIN ECDSA PRIVATE KEY-----
MHcCAQEEIEGZZ/4aD6tf0sc1ovyctlGWRSFp7RGw5ovRONZKLg4eoAoGCCqGSM49
AwEHoUQDQgAE1hO01GUQqH+FqstJ8ixLOIOQRDwTs4ntrOFMEmLVDH58Nrp0aD1/
++50YSLMnl31Mg1UtM/4l6bCDoUGFqOJUA==
-----END ECDSA PRIVATE KEY-----
`)
	block, _ := pem.Decode(ecKey)
	if _, err := ParseX509PrivateKey(block.Bytes); err != nil {
		t.Error(err)
	}

	pkcs1Key := []byte(`
-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQC/n0hrdlATHWkbIRPQxM0SqZM2jh2U6iJNz1UnWVWzWLHtsjMB
jBBxyK3T1PPcL1N6jwdozU3pSZ06GvR1lt2tBln1pCq87dGQLS5SRADFqMc2MysQ
R9+7bFxFV9sdSoNo5qmIqCn7HGP+1tVggmndyxKUZ8UEJeqlpPClolu4LQIDAQAB
AoGAZnngWNfk6tQHqaJ+3l7g7OITAFHwL/smbUY729dCpa8BOITcEi+9e5w+mdKA
t3f3xMtIwxtIV0Iu+yv/IAOWeEv36ZnWIUn2zrafc4Udb5HFFSJITaMYFePWQkGT
gdJGWBnUwKI8HZ4KVcpf51eATxQipr09ku7E8avpeFlaDgECQQDw/ckf0PcMmJ0z
CWEXlG4dYx0vGL4Qe2whpzHl15+ZacuH2ubMusn8RmrhYUhB61gLeTrrEh61qHUz
82Eg4SAdAkEAy45i9wActlTCQwZsfqIRMhHUmBqekKMtfP0jogEg+Rb3xEWWD5TT
FtMpzthp44cuW18vxBLm/Aiv0lFSvjubUQJACr6l+v8sJKmhSKdAZva9Oh4zLOhj
gQSfu5hixyan/QUUiAgghjxFyTOyjD30JMxDbq+HCvgR2nOaViihyf9/mQJAJx0Q
FSg3XC3DOj/UvyyDI1zbvfJ3f5FrXzYBD9Mps9Ne/p7ka9AME7m1seJRzn+eP751
njaHnexJZJ9tx9IKUQJAcM3JOT7qoQfKN3CkDI0CSgGxhPc63k8hT7EC795nwfmv
C6pigCN7rISVDB9mssOT3xHG4eTqot+ze5lj3+vsXg==
-----END RSA PRIVATE KEY-----
`)
	block, _ = pem.Decode(pkcs1Key)
	if _, err := ParseX509PrivateKey(block.Bytes); err != nil {
		t.Error(err)
	}

	pkcs8Key := []byte(`
-----BEGIN PRIVATE KEY-----
MIICdgIBADANBgkqhkiG9w0BAQEFAASCAmAwggJcAgEAAoGBAK5bg7HyKtW/H/GI
81tIcrTealpSXKxOfi+x4GCh4GE8B5X78UoTIXc8siDR11gg7ow/Sjz8oMuMn3Js
Z8xnX/96piUJXvyf9RtM36r6fTuj7nAXmk8WJ9UpIPemq0jIwumoeXDBR3/OG+5T
cKjjt5FBRmeBbuqRE5TjGXL6ehAfAgMBAAECgYAvjNOopuOZsWrzwyajIWnu+61D
fqy5bRqqxTbtA5iey/FBrRkL25XH3+ywWKoC5dBIvUXwxfDQMkSVvwy7yQF7xoBM
lH56ETZ4y5tFmOBN0zJbc/zNYSa2kRwKTVPHSGx6shKjjwRID1CV6EsDlZcIDYam
+h5l5r/zTWCJfkzcEQJBAOXn1Ap9hcvxBy8gD/1CNvlEL3jmk4vEpFzGcpTkkBHF
PWV+lF+TAuBDsuGJ0ZKObJ2KD/+QSG27DJG1PktFpBkCQQDCJawJkKiTz68UIvaO
SKiEkAC7sFGBiMsTrMIYfiMBRlyhb+p+ue4EZQM2pb5XyJhSHI70vKwtM86N+zB5
xhz3AkEAn8mBcQ0OqdDKSnGBS2by6EoAqImw8IpqJeTzDBLTItawNlNEciyt6gqD
UAfGdZKxYMjtF5VDEajYMokCU7SCOQJADO5PaA+verDRe2jcvRtPxgyFT/rtYtBG
nlfaUaFNGY+jKcl3d9tQZBCKR0CAzf35TzbBusE6QoR51HEPiHzOfwJAdhX4VL4G
T+LZUziv/XZHiPBtZ8y809D9B33BAz7kHwdFgPqIU9YcbFL6yTm2haIFb7eMSzEZ
sDxNsb5kHmOp8A==
-----END PRIVATE KEY-----
`)
	block, _ = pem.Decode(pkcs8Key)
	if _, err := ParseX509PrivateKey(block.Bytes); err != nil {
		t.Error(err)
	}
}
