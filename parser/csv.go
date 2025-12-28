package parser

import (
	"encoding/csv"
	"os"
)

// WriteHLTVCSV writes HLTV articles list to CSV
func WriteHLTVCSV(path string, articles []map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"id", "slug"}); err != nil {
		return err
	}

	for _, a := range articles {
		if err := w.Write([]string{a["id"], a["slug"]}); err != nil {
			return err
		}
	}
	return nil
}

// ReadHLTVCSV reads HLTV articles list from CSV
func ReadHLTVCSV(path string) ([]map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	var res []map[string]string
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 2 {
			continue
		}
		res = append(res, map[string]string{"id": row[0], "slug": row[1]})
	}
	return res, nil
}

// WriteCybersportCSV writes Cybersport articles list to CSV
func WriteCybersportCSV(path string, articles []map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"tag", "slug"}); err != nil {
		return err
	}

	for _, a := range articles {
		if err := w.Write([]string{a["tag"], a["slug"]}); err != nil {
			return err
		}
	}
	return nil
}

// ReadCybersportCSV reads Cybersport articles list from CSV
func ReadCybersportCSV(path string) ([]map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	var res []map[string]string
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 2 {
			continue
		}
		res = append(res, map[string]string{"tag": row[0], "slug": row[1]})
	}
	return res, nil
}
