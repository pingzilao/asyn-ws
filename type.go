package base

type ClearHanderType func(conId, qid string)
type PackPushHanderType func(reqster Requester, data interface{}) ([]byte, error)
type PackResHanderType func(reqster Requester, errCode int32, data interface{}) ([]byte, error)

