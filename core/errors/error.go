package terrr

import "errors"

// ErrWouldBlock は、非ブロッキング操作がすぐに完了できない場合に返されるエラー
var ErrWouldBlock = errors.New("operation would block")
