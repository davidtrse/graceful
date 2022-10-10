package tus

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

const dirName = "fileserver"

type fileHandler struct {
	db      *sql.DB
	dirPath string
}

type file struct {
	fileID         int
	offset         *int
	uploadLength   int
	uploadComplete *bool
}

func (fh fileHandler) File(fileID string) (file, error) {
	fID, err := strconv.Atoi(fileID)
	if err != nil {
		log.Println("Unable to convert fileID to string", err)
		return file{}, err
	}
	log.Println("going to query for fileID", fID)
	gfstmt := `select file_id, file_offset, file_upload_length, file_upload_complete from file where file_id = $1`
	row := fh.db.QueryRow(gfstmt, fID)
	f := file{}
	err = row.Scan(&f.fileID, &f.offset, &f.uploadLength, &f.uploadComplete)
	if err != nil {
		log.Println("error while fetching file", err)
		return file{}, err
	}
	return f, nil
}

func (fh fileHandler) fileDetailsHandler(e echo.Context) {
	fID := e.Param("tusID")
	file, err := fh.File(fID)
	if err != nil {
		e.Response().WriteHeader(http.StatusNotFound)
		return
	}
	log.Println("going to write upload offset to output")
	e.Response().Header().Set("Upload-Offset", strconv.Itoa(*file.offset))
	e.Response().WriteHeader(http.StatusOK)
	return
}

func (fh fileHandler) createFileHandler(e echo.Context) {
	ul, err := strconv.Atoi(e.Request().Header.Get("Upload-Length"))
	if err != nil {
		er := "Improper upload length"
		log.Printf("%s %s", er, err)
		e.Response().WriteHeader(http.StatusBadRequest)
		e.Response().Write([]byte(er))
		return
	}
	log.Printf("upload length %d\n", ul)
	io := 0
	uc := false
	f := file{
		offset:         &io,
		uploadLength:   ul,
		uploadComplete: &uc,
	}
	fileID, err := fh.createFile(f)
	if err != nil {
		er := "Error creating file in DB"
		log.Printf("%s %s\n", er, err)
		e.Response().WriteHeader(http.StatusInternalServerError)
		return
	}

	filePath := path.Join(fh.dirPath, fileID)
	file, err := os.Create(filePath)
	if err != nil {
		er := "Error creating file in filesystem"
		log.Printf("%s %s\n", er, err)
		e.Response().WriteHeader(http.StatusInternalServerError)
		return
	}
	defer file.Close()
	e.Response().Header().Set("Location", fmt.Sprintf("localhost:8080/files/%s", fileID))
	e.Response().WriteHeader(http.StatusCreated)
	return
}

func (fh fileHandler) createTable() error {
	q := `CREATE TABLE IF NOT EXISTS file(file_id SERIAL PRIMARY KEY, 
           file_offset INT NOT NULL, file_upload_length INT NOT NULL, file_upload_complete BOOLEAN NOT NULL, 
          created_at TIMESTAMP default NOW() NOT NULL, modified_at TIMESTAMP default NOW() NOT NULL)`
	_, err := fh.db.Exec(q)
	if err != nil {
		return err
	}
	log.Println("table create successfully")
	return nil
}

func (fh fileHandler) createFile(f file) (string, error) {
	cfstmt := `INSERT INTO file(file_offset, file_upload_length, file_upload_complete) VALUES($1, $2, $3) RETURNING file_id`
	fileID := 0
	err := fh.db.QueryRow(cfstmt, f.offset, f.uploadLength, f.uploadComplete).Scan(&fileID)
	if err != nil {
		return "", err
	}
	fid := strconv.Itoa(fileID)
	return fid, nil
}

func (fh fileHandler) updateFile(f file) error {
	var query []string
	var param []interface{}
	if f.offset != nil {
		of := fmt.Sprintf("file_offset = $1")
		ofp := f.offset
		query = append(query, of)
		param = append(param, ofp)
	}
	if f.uploadComplete != nil {
		uc := fmt.Sprintf("file_upload_complete = $2")
		ucp := f.uploadComplete
		query = append(query, uc)
		param = append(param, ucp)
	}

	if len(query) > 0 {
		mo := "modified_at = $3"
		mop := "NOW()"

		query = append(query, mo)
		param = append(param, mop)

		qj := strings.Join(query, ",")

		sqlq := fmt.Sprintf("UPDATE file SET %s WHERE file_id = $4", qj)

		param = append(param, f.fileID)

		log.Println("generated update query", sqlq)
		_, err := fh.db.Exec(sqlq, param...)

		if err != nil {
			log.Println("Error during file update", err)
			return err
		}
	}
	return nil
}

func createFileDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		log.Println("Error while fetching user home directory", err)
		return "", err
	}
	home := u.HomeDir
	dirPath := path.Join(home, dirName)
	err = os.MkdirAll(dirPath, 0744)
	if err != nil {
		log.Println("Error while creating file server directory", err)
		return "", err
	}
	return dirPath, nil
}
