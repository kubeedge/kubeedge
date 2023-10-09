package common

// DataModel defined standard data model for transferring data between interfaces
type DataModel struct {
	// TODO DataModel is standardized data, need add field
	DeviceName   string
	PropertyName string

	Value string
	Type  string

	TimeStamp int64
}

type Option func(*DataModel)

func (dm *DataModel) SetType(dataType string) {
	dm.Type = dataType
}

func (dm *DataModel) SetValue(data string) {
	dm.Value = data
}

func (dm *DataModel) SetTimeStamp() {
	dm.TimeStamp = getTimestamp()
}

func WithType(dataType string) Option {
	return func(model *DataModel) {
		model.Type = dataType
	}
}

func WithValue(data string) Option {
	return func(model *DataModel) {
		model.Value = data
	}
}

func WithTimeStamp(timeStamp int64) Option {
	return func(model *DataModel) {
		model.TimeStamp = timeStamp
	}
}

func NewDataModel(deviceName string, propertyName string, options ...Option) *DataModel {
	dataModel := &DataModel{
		DeviceName:   deviceName,
		PropertyName: propertyName,
		TimeStamp:    getTimestamp(),
	}
	for _, option := range options {
		option(dataModel)
	}
	return dataModel
}
