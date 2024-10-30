package driver;

import com.ghgande.j2mod.modbus.ModbusException;
import com.ghgande.j2mod.modbus.io.ModbusTCPTransaction;
import com.ghgande.j2mod.modbus.msg.ReadInputRegistersRequest;
import com.ghgande.j2mod.modbus.msg.ReadInputRegistersResponse;
import com.ghgande.j2mod.modbus.msg.ReadMultipleRegistersRequest;
import com.ghgande.j2mod.modbus.msg.ReadMultipleRegistersResponse;
import com.ghgande.j2mod.modbus.net.TCPMasterConnection;
import com.ghgande.j2mod.modbus.procimg.InputRegister;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import service.CustomizedClient_I;

import java.net.InetAddress;
import java.net.UnknownHostException;
import java.util.concurrent.locks.ReentrantLock;
@Slf4j
@Getter @Setter
public class CustomizedClient implements CustomizedClient_I {
    private ReentrantLock deviceMutex = new ReentrantLock();
    private CustomizedProtocolConfig protocolConfig;
    // TODO add some variables to help you better implement device drivers
    // Example: Modbus
    private ModbusTCPTransaction modbusTCPTransaction;

    public CustomizedClient(){}

    public CustomizedClient(CustomizedProtocolConfig protocolConfig){
        this.protocolConfig = protocolConfig;
        this.deviceMutex = new ReentrantLock();
        // TODO initialize the variables you added
        // Example: Modbus
        String[] address_port = protocolConfig.getConfigData().getAddress().split(":");
        String addr = address_port[0];
        String port = address_port[1];
        InetAddress Address = null;
        try {
            Address = InetAddress.getByName(addr);
        } catch (UnknownHostException e) {
            log.error("Unknown host: {}",protocolConfig.getConfigData().getAddress(),e);
        }
        TCPMasterConnection modbusConnection = new TCPMasterConnection(Address);
        modbusConnection.setPort(Integer.parseInt(port));
        try {
            modbusConnection.connect();
        } catch (Exception e) {
            log.error("Modbus device connection error: {}",e.getMessage(),e);
        }
        this.modbusTCPTransaction = new ModbusTCPTransaction(modbusConnection);
    }

    @Override
    public void initDevice() {
        // TODO: add init operation
    }

    @Override
    public Object getDeviceData(VisitorConfig visitorConfig) {
        // TODO: add the code to get device's data
        // Example: Modbus
        this.deviceMutex.lock();
        try {
            String registerType = visitorConfig.getVisitorConfigData().getRegister();
            int offset = visitorConfig.getVisitorConfigData().getOffset();

            if (registerType.equals("HoldingRegister")){
                ReadMultipleRegistersRequest request = new ReadMultipleRegistersRequest(0,offset);
                this.modbusTCPTransaction.setRequest(request);
                try {
                    this.modbusTCPTransaction.execute();
                    ReadMultipleRegistersResponse response = (ReadMultipleRegistersResponse) this.modbusTCPTransaction.getResponse();
                    InputRegister[] registers = response.getRegisters();
                    int result = registers[0].getValue();
                    return (short) result;
                } catch (ModbusException e) {
                    log.error("ModbusTCPTransaction execute error: {}",e.getMessage());
                }
            } else if (registerType.equals("InputRegister")) {
                ReadInputRegistersRequest request = new ReadInputRegistersRequest(0,offset);
                this.modbusTCPTransaction.setRequest(request);
                try {
                    this.modbusTCPTransaction.execute();
                    ReadInputRegistersResponse response = (ReadInputRegistersResponse) this.modbusTCPTransaction.getResponse();
                    InputRegister[] registers = response.getRegisters();
                    int result = registers[0].getValue();
                    return (short) result;
                } catch (ModbusException e) {
                    log.error("ModbusTCPTransaction execute error: {}",e.getMessage());
                }
            }else{
                log.warn("Unknown register type: {}", registerType);
            }
        }finally {
            this.deviceMutex.unlock();
        }
        return null;
    }

    @Override
    public void setDeviceData(Object data, VisitorConfig visitorConfig) {
        // TODO: set device's data
    }

    @Override
    public void stopDevice() {
        // TODO: add stop operation
    }
}
