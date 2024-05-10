package httpserver

import (
	"encoding/pem"
	"testing"
)

func TestParseX509PrivateKey(t *testing.T) {
	ecKey := []byte(`
-----BEGIN RSA PRIVATE KEY-----
MIIBOgIBAAJBAO58lcqkj5QZxwTAhBFeWGn1F8WuYtC65MDSN7Ec3mP+rG21Oq5s
KHGFdq0Hd15gXz+8JpSGfI1bAH1fuf5U2mcCAwEAAQJAAb1K4mV83pmph/FXhUDo
6CzvdXOmKNuUG4vs+A+714LaZ/19nhYXysGN8wNTzDnc9Tm5XJttZF9hnTqRsKh8
EQIhAP5n9MHz3OdQsqO/jws9H6zpauuByd5GeSwaS7wICzCTAiEA7/sYg6ql0DR0
L8/dvtiY92Cp0ezTHMcA5kT9AZbdF10CIQCEq6/faY487z+J138AbGrVYJDKqk+c
5AGS90+hikxTkwIgIZBO12twkXQko+NAskZ87mxYlAG5bRbwK0SO1kARYAkCIGHR
uXkEg1xCfzZmwtRK1QoQ0t5cLmZ3s4qu+nYJA9w4
-----END RSA PRIVATE KEY-----
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
	block, _ = pem.Decode(pkcs8Key)
	if _, err := ParseX509PrivateKey(block.Bytes); err != nil {
		t.Error(err)
	}
}
