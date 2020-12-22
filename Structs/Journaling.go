package Structs

type Journaling struct {
	Journal_type_operation [10]byte //type of operation to realize
	Journal_type           byte     //file or Folder
	Journal_name           [12]byte //name of file or Folder
	Journal_content        int64
	Journal_date           [19]byte //date of transaction
	Journal_property       [10]byte //property of file or Folder
	Journal_permits        int64    //permits to has the file or folder
}
