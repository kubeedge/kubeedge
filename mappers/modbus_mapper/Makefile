#make modbus_mapper
.PHONY: default modbus_mapper
modbus_mapper: 
	cd src && npm install --unsafe-perm=true
	docker build -t modbus_mapper:v1.0 .
