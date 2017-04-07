package fmp4

import (
	"aac"
	"flvFileReader"
	"fmt"
	"log"
	"mp3"
	"os"
	"strconv"
	"strings"
)

const (
	video_trak = 1
	audio_trak = 2
)
const (
	PCM_Platform_endian         = 0
	ADPCM                       = 1
	MP3                         = 2
	PCM_little_endian           = 3
	Nellymoser_16_mono          = 4
	Nellymoser_8_mono           = 5
	Nellymoser                  = 6
	G711_A_law_logarithmic_PCM  = 7
	G711_mu_law_logarithmic_PCM = 8
	AAC                         = 10
	Speex                       = 11
	MP3_8                       = 14
	Device_specific_sound       = 15
)

const (
	UserAgent_FireFox = "firefox"
	UserAgent_Android = "android"
)

type FMP4Slice struct {
	Data  []byte
	Idx   int  //0 base,-1 for init
	Video bool //audio or video
}

//cts:composition time
//dts:decode time:frame timestamp
//pts:Presentation  time
//pts=dts+cts,duraton base on pts

type FMP4Creater struct {
	videoIdx      int
	videoInited   bool
	videoLastTime uint32
	audioIdx      int
	audioInited   bool
	audioLastTime uint32

	width               int
	height              int
	fps                 int
	audioSampleSize     uint32
	audioSampleRate     uint32
	audioSampleDuration uint32
	ascData             []byte
	audioType           int
}

type FMP4Flags struct {
	IsLeading           uint32
	SampleDependsOn     uint32
	SampleIsDependedOn  uint32
	SampleHasRedundancy uint32
	IsAsync             uint32
}

func (this *FMP4Creater) AddFlvTag(tag *flvFileReader.FlvTag) (slice *FMP4Slice) {
	switch tag.TagType {
	case flvFileReader.FLV_TAG_ScriptData:
		return
	case flvFileReader.FLV_TAG_Audio:
		slice = this.handleAudioTag(tag)
		return
	case flvFileReader.FLV_TAG_Video:
		slice = this.handleVideoTag(tag)
		log.Println(tag.Timestamp)
		return
	default:
		return
	}
}

func (this *FMP4Creater) handleAudioTag(tag *flvFileReader.FlvTag) (slice *FMP4Slice) {
	if this.audioInited == false {
		this.audioInited = true
		return this.createAudioInitSeg(tag)
	} else {
		return this.createAudioSeg(tag)
	}
	return
}

func (this *FMP4Creater) handleVideoTag(tag *flvFileReader.FlvTag) (slice *FMP4Slice) {
	if tag.Data[0] != 0x17 && tag.Data[0] != 0x27 {
		log.Println(fmt.Sprintf("%d not support now", int(tag.Data[0])))
		return
	}
	pktType := tag.Data[1]
	//CompositionTime := 0
	cur := 2
	//	if pktType == 1 {
	//		CompositionTime = ((int(tag.Data[cur+0])) << 16) | ((int(tag.Data[cur+1])) << 8) | ((int(tag.Data[cur+2])) << 0)
	//		log.Println(CompositionTime)
	//	}
	cur += 3
	if this.videoInited == false {
		if pktType != 0 {
			log.Println("AVC pkt not find")
			return
		}
		this.videoInited = true
		return this.createVideoInitSeg(tag)
	} else {
		//one tag,one slice
		//one tag,may not one frame
		return this.createVideoSeg(tag)
	}
	return
}

func (this *FMP4Creater) createVideoInitSeg(tag *flvFileReader.FlvTag) (slice *FMP4Slice) {
	slice = &FMP4Slice{}
	slice.Video = true
	slice.Idx = -1
	segEncoder := flvFileReader.AMF0Encoder{}
	segEncoder.Init()
	//ftyp
	ftyp := &MP4Box{}
	ftyp.Push([]byte("ftyp"))
	ftyp.PushBytes([]byte("isom"))
	ftyp.Push4Bytes(1)
	ftyp.PushBytes([]byte("isom"))
	ftyp.PushBytes([]byte("avc1"))
	ftyp.Pop()
	err := segEncoder.AppendByteArray(ftyp.Flush())
	if err != nil {
		log.Println(err.Error())
		return
	}
	//moov
	moovBox := &MP4Box{}
	moovBox.Push([]byte("moov"))
	//mvhd
	duration := uint32(0)
	moovBox.Push([]byte("mvhd"))
	moovBox.Push4Bytes(0)        //version
	moovBox.Push4Bytes(0)        //creation_time
	moovBox.Push4Bytes(0)        //modification_time
	moovBox.Push4Bytes(1000)     //time_scale
	moovBox.Push4Bytes(duration) //duration 1s
	log.Println("duration 0xffffffff now")
	moovBox.Push4Bytes(0x00010000) //rate
	moovBox.Push2Bytes(0x0100)     //volume
	moovBox.Push2Bytes(0)          //reserved
	moovBox.Push8Bytes(0)          //reserved
	moovBox.Push4Bytes(0x00010000) //matrix
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0) //matrix
	moovBox.Push4Bytes(0x00010000)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0) //matrix
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x40000000)
	moovBox.Push4Bytes(0x0) //pre_defined
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	//nextrack id
	moovBox.Push4Bytes(0xffffffff)
	//!mvhd
	moovBox.Pop()
	//trak
	moovBox.Push([]byte("trak"))
	//tkhd
	moovBox.Push([]byte("tkhd"))
	moovBox.Push4Bytes(0x07) //version and flag
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(video_trak) //track id
	moovBox.Push4Bytes(0)          //reserved
	moovBox.Push4Bytes(duration)   //duration
	log.Println("duration 0xffffffff")
	moovBox.Push8Bytes(0)          //reserved
	moovBox.Push2Bytes(0)          //layer
	moovBox.Push2Bytes(0)          //alternate_group
	moovBox.Push2Bytes(0)          //volume
	moovBox.Push2Bytes(0)          //reserved
	moovBox.Push4Bytes(0x00010000) //matrix
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0) //matrix
	moovBox.Push4Bytes(0x00010000)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0) //matrix
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x40000000) //matrix
	//parse sps ,get w h fps
	tmpTagData := make([]byte, len(tag.Data))
	copy(tmpTagData, tag.Data)
	this.width, this.height, this.fps = flvFileReader.ParseSPS(tmpTagData[13:])
	moovBox.Push4Bytes(uint32(this.width << 16))  //width
	moovBox.Push4Bytes(uint32(this.height << 16)) //height
	//!tkhd
	moovBox.Pop()
	//mdia
	moovBox.Push([]byte("mdia"))
	//mdhd
	moovBox.Push([]byte("mdhd"))
	moovBox.Push4Bytes(0)        //version and flag
	moovBox.Push4Bytes(0)        //creation_time
	moovBox.Push4Bytes(0)        //modification_time
	moovBox.Push4Bytes(1000)     //time scale
	moovBox.Push4Bytes(duration) //duration
	log.Println("duration 0xffffffff")
	moovBox.Push4Bytes(0x55c40000) //language und
	//!mdhd
	moovBox.Pop()
	//hdlr
	moovBox.Push([]byte("hdlr"))
	moovBox.Push4Bytes(0) //version and flag
	moovBox.Push4Bytes(0) //reserved
	moovBox.PushBytes([]byte("vide"))
	moovBox.Push4Bytes(0) //reserved
	moovBox.Push4Bytes(0) //reserved
	moovBox.Push4Bytes(0) //reserved
	moovBox.PushBytes([]byte("VideoHandler"))
	moovBox.PushByte(0)
	//!hdlr
	moovBox.Pop()
	//minf
	moovBox.Push([]byte("minf"))
	//vmhd
	moovBox.Push([]byte("vmhd"))
	moovBox.Push4Bytes(1) //
	moovBox.Push2Bytes(0) //copy
	moovBox.Push2Bytes(0) //opcolor
	moovBox.Push2Bytes(0) //opcolor
	moovBox.Push2Bytes(0) //opcolor
	//!vmhd
	moovBox.Pop()
	//dinf
	moovBox.Push([]byte("dinf"))
	//dref
	moovBox.Push([]byte("dref"))
	moovBox.Push4Bytes(0) //version
	moovBox.Push4Bytes(1) //entry_count
	//url
	moovBox.Push([]byte("url "))
	moovBox.Push4Bytes(1)
	//!url
	moovBox.Pop()
	//!dref
	moovBox.Pop()
	//!dinf
	moovBox.Pop()
	//stbl
	moovBox.Push([]byte("stbl"))
	this.stsdV(moovBox, tag) //stsd
	//stts
	moovBox.Push([]byte("stts"))
	moovBox.Push4Bytes(0) //version
	moovBox.Push4Bytes(0) //count
	//!stts
	moovBox.Pop()
	//stsc
	moovBox.Push([]byte("stsc"))
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	//!stsc
	moovBox.Pop()
	//stsz
	moovBox.Push([]byte("stsz"))
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	//!stsz
	moovBox.Pop()
	//stco
	moovBox.Push([]byte("stco"))
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	//!stco
	moovBox.Pop()
	//!stbl
	moovBox.Pop()
	//!minf
	moovBox.Pop()
	//!mdia
	moovBox.Pop()
	//!trak
	moovBox.Pop()
	//mvex
	moovBox.Push([]byte("mvex"))
	//trex
	moovBox.Push([]byte("trex"))
	moovBox.Push4Bytes(0)          //version and flag
	moovBox.Push4Bytes(video_trak) //track id
	moovBox.Push4Bytes(1)          //default_sample_description_index
	moovBox.Push4Bytes(0)          //default_sample_duration
	moovBox.Push4Bytes(0)          //default_sample_size
	moovBox.Push4Bytes(0x00010001) //default_sample_flags
	//!trex
	moovBox.Pop()
	//!mvex
	moovBox.Pop()
	//!moov
	moovBox.Pop()

	err = segEncoder.AppendByteArray(moovBox.Flush())
	if err != nil {
		log.Println(err.Error())
		return
	}
	slice.Data, err = segEncoder.GetData()
	if err != nil {
		log.Println(err.Error())
		return
	}

	fp, err := os.OpenFile("video/init.mp4", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer fp.Close()
	fp.Write(slice.Data)
	return
}

func (this *FMP4Creater) createVideoSeg(tag *flvFileReader.FlvTag) (slice *FMP4Slice) {
	slice = &FMP4Slice{}
	slice.Video = true
	slice.Idx = this.videoIdx
	this.videoIdx++
	segEncoder := flvFileReader.AMF0Encoder{}
	segEncoder.Init()

	flags := &FMP4Flags{}
	flags.IsLeading = 0
	flags.SampleHasRedundancy = 0

	if tag.Data[0] == 0x17 {
		flags.SampleDependsOn = 2
		flags.SampleIsDependedOn = 1
		flags.IsAsync = 0
	} else if tag.Data[0] == 0x27 {
		flags.SampleDependsOn = 1
		flags.SampleIsDependedOn = 0
		flags.IsAsync = 1
	} else {
		log.Fatal("invalid video")
	}

	videBox := &MP4Box{}
	//moof
	videBox.Push([]byte("moof"))
	//mfhd
	videBox.Push([]byte("mfhd"))
	videBox.Push4Bytes(0) //version and flags
	videBox.Push4Bytes(uint32(this.videoIdx))
	//mfhd
	videBox.Pop()
	//traf
	videBox.Push([]byte("traf"))
	//tfhd
	videBox.Push([]byte("tfhd"))
	videBox.Push4Bytes(0)          //version and flags,no default-base-is-moof
	videBox.Push4Bytes(video_trak) //track
	//!tfhd
	videBox.Pop()
	//tfdt
	videBox.Push([]byte("tfdt"))
	videBox.Push4Bytes(0)
	videBox.Push4Bytes(tag.Timestamp)
	//!tfdt
	videBox.Pop()
	//trun
	videBox.Push([]byte("trun"))
	videBox.Push4Bytes(0xf01) //offset,duration,samplesize,composition
	videBox.Push4Bytes(1)     //1 sample
	videBox.Push4Bytes(0x79)  //offset:if base-is-moof ,data offset,from moov begin to mdat data,so now base is first byte
	if tag.Timestamp-this.videoLastTime == 0 {
		//no duration,just a first frame
		videBox.Push4Bytes(uint32(1000 / this.fps)) //duration
		//log.Println(uint32(1000 / this.fps))
	} else {
		videBox.Push4Bytes(tag.Timestamp - this.videoLastTime) //duration
		//log.Println(tag.Timestamp - this.videoLastTime)
		//log.Println(this.videoLastTime)
	}
	composition := (uint32(tag.Data[2]) << 16) | (uint32(tag.Data[3]) << 8) | (uint32(tag.Data[4]) << 0)
	//log.Println(fmt.Sprintf("timestame:%d  composition:%d duration:%d", tag.Timestamp, composition, tag.Timestamp-this.videoLastTime))
	this.videoLastTime = tag.Timestamp
	videBox.Push4Bytes(uint32(len(tag.Data) - (5))) //sample size,mdat data size
	videBox.PushByte(uint8((flags.IsLeading << 2) | flags.SampleDependsOn))
	videBox.PushByte(uint8((flags.SampleIsDependedOn << 6) | (flags.SampleHasRedundancy << 4) | flags.IsAsync))
	videBox.Push2Bytes(0)
	videBox.Push4Bytes(composition) //sample_composition_time
	//!trun
	videBox.Pop()
	//sdtp
	videBox.Push([]byte("sdtp"))
	videBox.Push4Bytes(0)
	videBox.PushByte(uint8((flags.IsLeading << 6) | (flags.SampleDependsOn << 4) | (flags.SampleIsDependedOn << 2) | (flags.SampleHasRedundancy)))
	//!sdtp
	videBox.Pop()
	//!traf
	videBox.Pop()
	//!moof
	videBox.Pop()
	err := segEncoder.AppendByteArray(videBox.Flush())
	if err != nil {
		log.Println(err.Error())
		return
	}

	//mdat
	err = segEncoder.EncodeInt32(int32(len(tag.Data) - (5) + 8))
	if err != nil {
		log.Println(err.Error())
		return
	}
	err = segEncoder.AppendByteArray([]byte("mdat"))
	if err != nil {
		log.Println(err.Error())
		return
	}
	err = segEncoder.AppendByteArray(tag.Data[5:])
	//!mdat
	slice.Data, err = segEncoder.GetData()
	if err != nil {
		log.Println(err.Error())
		return
	}
	fileName := "video/segment_" + strconv.Itoa(slice.Idx) + ".m4s"
	//fp, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0666)
	fp, err := os.Create(fileName)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer fp.Close()
	fp.Write(slice.Data)
	return
}

func (this *FMP4Creater) createAudioInitSeg(tag *flvFileReader.FlvTag) (slice *FMP4Slice) {

	this.audioType = int(tag.Data[0] >> 4)
	log.Println(tag.Data)
	switch this.audioType {
	case MP3:
		this.audioSampleSize = 1152
		mp3Header, _ := mp3.ParseMP3Header(tag.Data[1:])
		if mp3Header != nil {
			this.audioSampleRate = uint32(mp3Header.SampleRate)
		}
	case AAC:
		this.audioSampleSize = 1024
		asc := aac.GenerateAudioSpecificConfig(tag.Data[2:])
		this.audioSampleRate = uint32(asc.SamplingFrequency)
		log.Println(asc.AudioObjectType)
		log.Println(this.audioSampleRate)
		//		soundRate := ((tag.Data[0] & 0xC) >> 2)
		mpeg4Asc := aac.MP4AudioGetConfig(tag.Data[2:])
		log.Println(mpeg4Asc)
		if mpeg4Asc.Ext_object_type != 0 {
			this.audioSampleRate = uint32(mpeg4Asc.Ext_sample_rate)
		} else {
			this.audioSampleRate = uint32(mpeg4Asc.Sample_rate)
		}
		this.audioSampleDuration = this.audioSampleSize * 1000 / this.audioSampleRate
		//		switch soundRate {
		//		case 0:
		//			this.audioSampleDuration = this.audioSampleSize * 1000 / 5500
		//		case 1:
		//			this.audioSampleDuration = this.audioSampleSize * 1000 / 11000
		//		case 2:
		//			this.audioSampleDuration = this.audioSampleSize * 1000 / 22000
		//		case 3:
		//			this.audioSampleDuration = this.audioSampleSize * 1000 / 44000
		//		}
		log.Println(this.audioSampleDuration)
		log.Println(this.audioSampleRate)
		if mpeg4Asc.Ext_object_type == 0 {
			this.ascData = tag.Data[2:]
		} else {
			this.ascData = this.aacForHttp(tag, "")
		}
	default:
		log.Fatal("unknown audio type")
	}
	log.Println(this.audioSampleDuration)
	slice = &FMP4Slice{}
	slice.Video = true
	slice.Idx = -1
	segEncoder := flvFileReader.AMF0Encoder{}
	segEncoder.Init()
	//ftyp
	ftyp := &MP4Box{}
	ftyp.Push([]byte("ftyp"))
	ftyp.PushBytes([]byte("isom"))
	ftyp.Push4Bytes(1)
	ftyp.PushBytes([]byte("isom"))
	ftyp.PushBytes([]byte("avc1"))
	ftyp.Pop()
	err := segEncoder.AppendByteArray(ftyp.Flush())
	if err != nil {
		log.Println(err.Error())
		return
	}
	duration := uint32(0)
	//moov
	moovBox := &MP4Box{}
	moovBox.Push([]byte("moov"))
	//mvhd
	moovBox.Push([]byte("mvhd"))
	moovBox.Push4Bytes(0)        //version
	moovBox.Push4Bytes(0)        //creation_time
	moovBox.Push4Bytes(0)        //modification_time
	moovBox.Push4Bytes(1000)     //time_scale
	moovBox.Push4Bytes(duration) //duration 1s
	log.Println("duration 0 now")
	moovBox.Push4Bytes(0x00010000) //rate
	moovBox.Push2Bytes(0x0100)     //volume
	moovBox.Push2Bytes(0)          //reserved
	moovBox.Push8Bytes(0)          //reserved
	moovBox.Push4Bytes(0x00010000) //matrix
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0) //matrix
	moovBox.Push4Bytes(0x00010000)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0) //matrix
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x40000000)
	moovBox.Push4Bytes(0x0) //pre_defined
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	//nextrack id
	moovBox.Push4Bytes(0xffffffff)
	//!mvhd
	moovBox.Pop()
	//trak
	moovBox.Push([]byte("trak"))
	//tkhd
	moovBox.Push([]byte("tkhd"))
	moovBox.Push4Bytes(0x07) //version and flag
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(audio_trak) //track id
	moovBox.Push4Bytes(0)          //reserved
	moovBox.Push4Bytes(duration)   //duration
	log.Println("duration 0xffffffff")
	moovBox.Push8Bytes(0) //reserved
	moovBox.Push2Bytes(0) //layer
	moovBox.Push2Bytes(0) //alternate_group
	//moovBox.Push2Bytes(0x0100)     //volume
	moovBox.Push2Bytes(0)          //??
	moovBox.Push2Bytes(0)          //reserved
	moovBox.Push4Bytes(0x00010000) //matrix
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0) //matrix
	moovBox.Push4Bytes(0x00010000)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0) //matrix
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x40000000) //matrix
	moovBox.Push4Bytes(0)          //width
	moovBox.Push4Bytes(0)          //height
	//!tkhd
	moovBox.Pop()
	//mdia
	moovBox.Push([]byte("mdia"))
	//mdhd
	moovBox.Push([]byte("mdhd"))
	moovBox.Push4Bytes(0) //version and flag
	moovBox.Push4Bytes(0) //creation_time
	moovBox.Push4Bytes(0) //modification_time
	log.Println("maybe to audio sample hz,now use Video time")
	moovBox.Push4Bytes(1000)     //time scale
	moovBox.Push4Bytes(duration) //duration
	log.Println("duration 0xffffffff")
	if this.audioType == MP3 {
		moovBox.Push4Bytes(0x55c40000)
	} else {
		moovBox.Push4Bytes(0x55c40000) //language und
	}
	//!mdhd
	moovBox.Pop()
	//hdlr
	moovBox.Push([]byte("hdlr"))
	moovBox.Push4Bytes(0) //version and flag
	moovBox.Push4Bytes(0) //reserved
	moovBox.PushBytes([]byte("soun"))
	moovBox.Push4Bytes(0) //reserved
	moovBox.Push4Bytes(0) //reserved
	moovBox.Push4Bytes(0) //reserved
	moovBox.PushBytes([]byte("SoundHandler"))
	moovBox.PushByte(0)
	//!hdlr
	moovBox.Pop()
	//minf
	moovBox.Push([]byte("minf"))
	//smhd
	moovBox.Push([]byte("smhd"))
	moovBox.Push4Bytes(0) //version and flag
	moovBox.Push2Bytes(0) //balance
	moovBox.Push2Bytes(0) //reserved
	//!smhd
	moovBox.Pop()
	//dinf
	moovBox.Push([]byte("dinf"))
	//dref
	moovBox.Push([]byte("dref"))
	moovBox.Push4Bytes(0) //version
	moovBox.Push4Bytes(1) //entry_count
	//url
	moovBox.Push([]byte("url "))
	moovBox.Push4Bytes(1)
	//!url
	moovBox.Pop()
	//!dref
	moovBox.Pop()
	//!dinf
	moovBox.Pop()
	//stbl
	moovBox.Push([]byte("stbl"))
	this.stsdA(moovBox, tag) //stsd
	//stts
	moovBox.Push([]byte("stts"))
	moovBox.Push4Bytes(0) //version
	moovBox.Push4Bytes(0) //count
	//!stts
	moovBox.Pop()
	//stsc
	moovBox.Push([]byte("stsc"))
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	//!stsc
	moovBox.Pop()
	//stsz
	moovBox.Push([]byte("stsz"))
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	//!stsz
	moovBox.Pop()
	//stco
	moovBox.Push([]byte("stco"))
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	//!stco
	moovBox.Pop()
	//!stbl
	moovBox.Pop()
	//!minf
	moovBox.Pop()
	//!mdia
	moovBox.Pop()
	//!trak
	moovBox.Pop()
	//mvex
	moovBox.Push([]byte("mvex"))
	//trex
	moovBox.Push([]byte("trex"))
	moovBox.Push4Bytes(0)          //version and flag
	moovBox.Push4Bytes(audio_trak) //track id
	moovBox.Push4Bytes(1)          //default_sample_description_index
	moovBox.Push4Bytes(0)          //default_sample_duration
	moovBox.Push4Bytes(0)          //default_sample_size
	moovBox.Push4Bytes(0x00010001) //default_sample_flags
	//!trex
	moovBox.Pop()
	//!mvex
	moovBox.Pop()
	//!moov
	moovBox.Pop()

	err = segEncoder.AppendByteArray(moovBox.Flush())
	if err != nil {
		log.Println(err.Error())
		return
	}
	slice.Data, err = segEncoder.GetData()
	if err != nil {
		log.Println(err.Error())
		return
	}
	//	log.Println(slice)
	fp, err := os.Create("audio/init.mp4")
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer fp.Close()
	fp.Write(slice.Data)
	return
}

func (this *FMP4Creater) createAudioSeg(tag *flvFileReader.FlvTag) (slice *FMP4Slice) {
	slice = &FMP4Slice{}
	slice.Video = false
	slice.Idx = this.audioIdx
	this.audioIdx++
	segEncoder := flvFileReader.AMF0Encoder{}
	segEncoder.Init()

	sounBox := &MP4Box{}
	//moof
	sounBox.Push([]byte("moof"))
	//mfhd
	sounBox.Push([]byte("mfhd"))
	sounBox.Push4Bytes(0) //version and flags
	sounBox.Push4Bytes(uint32(this.audioIdx))
	//mfhd
	sounBox.Pop()
	//traf
	sounBox.Push([]byte("traf"))
	//tfhd
	sounBox.Push([]byte("tfhd"))
	sounBox.Push4Bytes(0)          //version and flags,no default-base-is-moof
	sounBox.Push4Bytes(audio_trak) //track
	//!tfhd
	sounBox.Pop()
	//tfdt
	sounBox.Push([]byte("tfdt"))
	sounBox.Push4Bytes(0)
	sounBox.Push4Bytes(this.audioLastTime)
	//!tfdt
	sounBox.Pop()
	//trun

	dataPrefixLength := 1
	if this.audioType == AAC {
		dataPrefixLength = 2
	} else if this.audioType == MP3 {
		dataPrefixLength = 1
	} else {
		log.Fatal("wth")
	}
	sounBox.Push([]byte("trun"))
	sounBox.Push4Bytes(0xf01) //offset,duration,samplesize,composition
	sounBox.Push4Bytes(1)     //1 sample
	sounBox.Push4Bytes(0x79)  //offset:if base-is-moof ,data offset,from moov begin to mdat data,so now base is first byte

	if tag.Timestamp-this.audioLastTime == 0 {
		//no duration,just a first frame
		sounBox.Push4Bytes(this.audioSampleDuration) //duration
	} else {
		//sounBox.Push4Bytes(this.audioSampleDuration)
		sounBox.Push4Bytes(tag.Timestamp - this.audioLastTime)
	}
	//log.Println(fmt.Sprintf("%d %d", this.audioLastTime, tag.Timestamp))
	//this.audioLastTime += this.audioSampleDuration
	this.audioLastTime = tag.Timestamp

	sounBox.Push4Bytes(uint32(len(tag.Data) - dataPrefixLength)) //sample size
	flags := &FMP4Flags{}
	flags.SampleDependsOn = 1
	sounBox.PushByte(uint8((flags.IsLeading << 2) | flags.SampleDependsOn))
	sounBox.PushByte(uint8((flags.SampleIsDependedOn << 6) | (flags.SampleHasRedundancy << 4) | flags.IsAsync))
	sounBox.Push2Bytes(0)
	sounBox.Push4Bytes(0) //sample_composition_time                                     //sample_composition_time
	//!trun
	sounBox.Pop()
	//sdtp
	sounBox.Push([]byte("sdtp"))
	sounBox.Push4Bytes(0)
	sounBox.PushByte(uint8((flags.IsLeading << 6) | (flags.SampleDependsOn << 4) | (flags.SampleIsDependedOn << 2) | (flags.SampleHasRedundancy)))
	//!sdtp
	sounBox.Pop()
	//!traf
	sounBox.Pop()
	//!moof
	sounBox.Pop()
	err := segEncoder.AppendByteArray(sounBox.Flush())
	if err != nil {
		log.Println(err.Error())
		return
	}

	//mdat
	err = segEncoder.EncodeInt32(int32(len(tag.Data) - dataPrefixLength + 8))
	if err != nil {
		log.Println(err.Error())
		return
	}
	err = segEncoder.AppendByteArray([]byte("mdat"))
	if err != nil {
		log.Println(err.Error())
		return
	}
	err = segEncoder.AppendByteArray(tag.Data[dataPrefixLength:])
	//!mdat
	slice.Data, err = segEncoder.GetData()
	if err != nil {
		log.Println(err.Error())
		return
	}

	fileName := "audio/segment_" + strconv.Itoa(slice.Idx) + ".m4s"
	fp, err := os.Create(fileName)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer fp.Close()
	fp.Write(slice.Data)
	//log.Println(slice.Idx)
	return
}

func (this *FMP4Creater) stsdV(box *MP4Box, tag *flvFileReader.FlvTag) {
	//stsd
	box.Push([]byte("stsd"))
	box.Push4Bytes(0)
	box.Push4Bytes(1)
	//avc1
	box.Push([]byte("avc1"))
	box.Push4Bytes(0)
	box.Push2Bytes(0)
	box.Push2Bytes(1)
	box.Push2Bytes(0)
	box.Push2Bytes(0)
	box.Push4Bytes(0)
	box.Push4Bytes(0)
	box.Push4Bytes(0)
	box.Push2Bytes(uint16(this.width))
	box.Push2Bytes(uint16(this.height))
	box.Push4Bytes(0x00480000)
	box.Push4Bytes(0x00480000)
	box.Push4Bytes(0)
	box.Push2Bytes(1)
	box.PushByte(uint8(len("fmp4 coding")))
	box.PushBytes([]byte("fmp4 coding"))
	spaceEnd := make([]byte, 32-1-len("fmp4 coding"))
	box.PushBytes(spaceEnd)
	box.Push2Bytes(0x18)
	box.Push2Bytes(0xffff)
	//avcC
	box.Push([]byte("avcC"))
	box.PushBytes(tag.Data[5:])
	//!avcC
	box.Pop()
	//!avc1
	box.Pop()
	//!stsd
	box.Pop()
	return
}

func (this *FMP4Creater) stsdA(box *MP4Box, tag *flvFileReader.FlvTag) {
	//stsd
	box.Push([]byte("stsd"))
	box.Push4Bytes(0)
	box.Push4Bytes(1)
	//mp4a
	box.Push([]byte("mp4a"))
	box.Push4Bytes(0)  //reserved
	box.Push2Bytes(0)  //reserved
	box.Push2Bytes(1)  //data reference index
	box.Push8Bytes(0)  //reserved int32[2]
	box.Push2Bytes(2)  //channel count
	box.Push2Bytes(16) //sample size
	box.Push2Bytes(0)  //pre defined
	box.Push2Bytes(0)  //reserved
	log.Println(this.audioSampleRate)
	box.Push4Bytes(this.audioSampleRate << 16) //samplerate
	//esds
	box.Push([]byte("esds"))
	box.Push4Bytes(0) //version and flag
	box.PushByte(3)   //tag
	esd := &MP4Box{}
	esd.Push2Bytes(0) //ES ID
	esd.PushByte(0)   //1:streamDependenceFlag=0  1:URL_Flag=0 1:OCRstreamFlag=0 5:streamPrority=0
	esd.PushByte(4)   //DecoderConfigDescriptor tag
	esdDesc := &MP4Box{}
	switch this.audioType { //object type indication
	case MP3:
		esdDesc.PushByte(0x6b)
	case AAC:
		esdDesc.PushByte(0x40)
	default:
		esdDesc.PushByte(0x40)
		log.Println(fmt.Sprintf("audio type %d not support", this.audioType))
	}
	esdDesc.PushByte(0x15) //固定15  streamType upstream reserved
	esdDesc.PushByte(0)    //24位buffer size db
	esdDesc.Push2Bytes(0)  //24位补充
	esdDesc.Push4Bytes(0)  //max bitrate
	esdDesc.Push4Bytes(0)  //avg bitrate
	if this.audioType == AAC {
		esdDesc.PushByte(0x05)
		if len(tag.Data) >= 2 {
			//esdDesc.PushByte(byte(len(tag.Data) - 2))
			//esdDesc.PushBytes(tag.Data[2:])
			//ascData := this.aacForHttp(tag, "")
			//log.Println(ascData)
			//esdDesc.PushByte(byte(len(ascData)))
			//esdDesc.PushBytes(ascData)
			esdDesc.PushByte(byte(len(this.ascData)))
			esdDesc.PushBytes(this.ascData)
		}

	}
	esdDescData := esdDesc.Flush()
	esd.PushByte(byte(len(esdDescData)))
	esd.PushBytes(esdDescData)
	esd.PushByte(0x06) //SLConfigDescrTag
	esd.PushByte(0x01) //length field
	esd.PushByte(0x02) //predefined 0x02 reserved for use int mp4 faile
	esdData := esd.Flush()
	box.PushByte(byte(len(esdData)))
	box.PushBytes(esdData)
	//!esds
	box.Pop()
	//!mp4a
	box.Pop()
	//!stsd
	box.Pop()
	return
}

func (this *FMP4Creater) aacForHttp(tag *flvFileReader.FlvTag, useragent string) (cfg []byte) {
	asc := aac.GenerateAudioSpecificConfig(tag.Data[2:])

	if len(useragent) > 0 {
		useragent = strings.ToLower(useragent)
	}
	switch useragent {
	case UserAgent_FireFox:
		if asc.SamplingFrequencyIndex >= aac.AAC_SCALABLE {
			asc.AudioObjectType = aac.AAC_HE_OR_SBR
			asc.ExtensionSamplingIndex = asc.SamplingFrequencyIndex - 3
			cfg = make([]byte, 4)
		} else {
			asc.AudioObjectType = aac.AAC_LC
			asc.ExtensionSamplingIndex = asc.SamplingFrequencyIndex
			cfg = make([]byte, 2)
		}
	case UserAgent_Android:
		asc.AudioObjectType = aac.AAC_LC
		asc.ExtensionSamplingIndex = asc.SamplingFrequencyIndex
		cfg = make([]byte, 2)
	default:
		asc.AudioObjectType = aac.AAC_HE_OR_SBR
		asc.ExtensionSamplingIndex = asc.SamplingFrequencyIndex
		cfg = make([]byte, 4)
		if asc.SamplingFrequencyIndex >= aac.AAC_SCALABLE {
			asc.ExtensionSamplingIndex = asc.SamplingFrequencyIndex - 3
		} else if asc.ChannelConfiguration == 1 {
			asc.AudioObjectType = aac.AAC_LC
			asc.ExtensionSamplingIndex = asc.SamplingFrequencyIndex
			cfg = make([]byte, 2)
		}
	}
	cfg[0] = (asc.AudioObjectType << 3)
	cfg[0] |= ((asc.SamplingFrequencyIndex & 0xf) >> 1)
	cfg[1] = ((asc.SamplingFrequencyIndex & 0xf) << 7)
	cfg[1] |= ((asc.ChannelConfiguration & 0xf) << 3)
	if asc.AudioObjectType == aac.AAC_HE_OR_SBR {
		cfg[1] |= ((asc.ExtensionSamplingIndex & 0xf) >> 1)
		cfg[2] = ((asc.ExtensionSamplingIndex & 1) << 7)
		cfg[2] |= (2 << 2)
		cfg[3] = 0
	}
	return
}
