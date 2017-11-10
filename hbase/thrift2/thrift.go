package thrift2

import (
	"github.com/widaT/golib/hbase"
	"git.apache.org/thrift.git/lib/go/thrift"
)

type HClient struct {
	c *hbase.THBaseServiceClient
	t *thrift.TSocket
}

func New(host string,port string) (c *HClient ,err error) {
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTSocket(host + ":" + port)
	if err != nil {
		return
	}
	client := hbase.NewTHBaseServiceClientFactory(transport, protocolFactory)
	if err = transport.Open(); err != nil {
		return
	}

	c = &HClient{c:client,t:transport}
	return
}


func (h *HClient)Close() {
	 h.t.Close()
}

func (h *HClient)Exist(table,rowkey []byte)( bool, error) {
	return  h.c.Exists(table, &hbase.TGet{Row: rowkey})
}

func (h *HClient)Put(table,rowkey,family,qualifier,value []byte) error {
	cvarr := []*hbase.TColumnValue{{
			Family: family,
			Qualifier: qualifier,
			Value: value,
		},
	}
	temptput := hbase.TPut{Row: rowkey, ColumnValues: cvarr}
	err := h.c.Put([]byte(table), &temptput)
	if err != nil {
		return err
	}
	return nil
}


func (h *HClient)Get(table []byte, tget *hbase.TGet) (*hbase.TResult_, error) {
	return h.c.Get(table, tget)
}


// when key not exists, the value is nil
func (h *HClient)GetKeyValues(table, family,rowKey,qualifier,filter []byte) (*hbase.TResult_, error) {
	tget := &hbase.TGet{
		Row: rowKey,
		Columns: []*hbase.TColumn{
			&hbase.TColumn{
				Family: family,
				Qualifier : qualifier,
			},
		},
		//FilterString:[]byte("ColumnPrefixFilter('f')"),
		FilterString:filter,
	}
	return h.c.Get(table, tget)
}

// when key not exists, the value is nil
func (h *HClient)GetKeySingleValue(table, family, rowKey, qualifier []byte) (rowValue []byte, err error) {
	tget := &hbase.TGet{
		Row: rowKey,
		Columns: []*hbase.TColumn{
			&hbase.TColumn{
				Family:    family,
				Qualifier: qualifier,
			},
		},
	}
	if tresult, e := h.c.Get(table, tget); e != nil {
		err = e
		return
	} else {
		if 0 == len(tresult.ColumnValues) {
			return
		} else {
			rowValue = tresult.ColumnValues[0].Value
			return
		}
	}
}

func (h *HClient)Detele(table,rowkey []byte) error {
	tdelete := hbase.TDelete{Row: []byte(rowkey)}
	return  h.c.DeleteSingle(table, &tdelete)
}


func (h *HClient)PutMultiple(table []byte,tput []*hbase.TPut) error {
	return  h.c.PutMultiple(table, tput)
}


func (h *HClient)GetMultiple(table []byte,tget []*hbase.TGet) ( []*hbase.TResult_,  error) {
	return  h.c.GetMultiple(table, tget)
}

func (h *HClient)DelMultiple(table []byte,tdel[]*hbase.TDelete) ( []*hbase.TDelete,  error) {
	return  h.c.DeleteMultiple(table, tdel)
}

func (h *HClient)OpenScannerSimple(table,startrow,stoprow []byte, columns []*hbase.TColumn) (r int32, err error) {
	return  h.c.OpenScanner(table, &hbase.TScan{
		StartRow: startrow,
		StopRow: stoprow,
		// FilterString: []byte("RowFilter(=, 'regexstring:00[1-3]00')"),
		// FilterString: []byte("PrefixFilter('1407658495588-')"),
		Columns: columns,
	})
}

func (h *HClient)OpenScanner(table []byte, tscan *hbase.TScan) (r int32, err error) {
	return  h.c.OpenScanner(table,tscan)
}

func (h *HClient)GetScannerRows(scanresultnum int32,numRows int32) ( []*hbase.TResult_, error) {
	return  h.c.GetScannerRows(scanresultnum, numRows)
}

func (h *HClient)GetScannerResults(table,startrow,stoprow []byte, columns []*hbase.TColumn,numRows int32)  ([]*hbase.TResult_, error) {
	return  h.c.GetScannerResults(table, &hbase.TScan{
		StartRow: startrow,
		StopRow: stoprow,
		// FilterString: []byte("RowFilter(=, 'regexstring:00[1-3]00')"),
		// FilterString: []byte("PrefixFilter('1407658495588-')"),
		Columns: columns}, numRows)
}

func (h *HClient)CloseScanner(scanresultnum int32)  error {
	return h.c.CloseScanner(scanresultnum)
}
