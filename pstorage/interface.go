package pstorage

type PersistentStorage interface {
	Save(data interface{}) error
	Load(data interface{}) (bool, error)
}
