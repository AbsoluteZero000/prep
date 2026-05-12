package resume

import "errors"

var (
	ErrFileNotFound      = errors.New("resume file not found")
	ErrEmptyResume       = errors.New("resume file is empty")
	ErrUnreadableContent = errors.New("resume content could not be extracted")
	ErrUnsupportedFormat = errors.New("unsupported resume format")
	ErrNotAResume        = errors.New("file does not appear to be a resume")
	ErrResumeTooShort    = errors.New("resume has too little content")
)
