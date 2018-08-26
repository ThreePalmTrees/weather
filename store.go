package main

import (
	proto "github.com/golang/protobuf/proto"
	"github.com/jackdoe/go-metno"
	. "github.com/jackdoe/weather/log"
	pb "github.com/jackdoe/weather/spec"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"time"
)

const precision = 3

type store struct {
	db *leveldb.DB
}

func NewStore(path string) *store {

	db, err := leveldb.OpenFile(path, &opt.Options{
		BlockSize:           512 * opt.KiB,
		CompactionTableSize: 10 * opt.MiB,
		Compression:         opt.SnappyCompression,
		NoSync:              true,
	})

	if err != nil {
		panic(err)
	}

	return &store{
		db: db,
	}
}

func (s *store) close() {
	s.db.Close()
}

func (s *store) normalizeWeatherKey(k *pb.WeatherStoreKey) {
	if k.Timestamp == 0 {
		k.Timestamp = currentHour() + 3600
	}

	k.Lat = float64(int64(k.Lat + 0.5))
	k.Lng = float64(int64(k.Lng + 0.5))
}

func (s *store) getStoredWeather(k *pb.WeatherStoreKey) (*pb.WeatherStoreValue, error) {
	log := Log()
	log.Infof("AAA %#v", k)
	dataK, err := proto.Marshal(k)
	if err != nil {
		return nil, err
	}

	dataV, err := s.db.Get(dataK, nil)
	if err != nil {
		return nil, err
	}
	if dataK == nil {
		return nil, nil
	}

	out := &pb.WeatherStoreValue{}
	err = proto.Unmarshal(dataV, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *store) setStoredWeather(k *pb.WeatherStoreKey, v *pb.WeatherStoreValue) error {
	dataK, err := proto.Marshal(k)
	if err != nil {
		return err
	}

	dataV, err := proto.Marshal(v)
	if err != nil {
		return err
	}

	err = s.db.Put(dataK, dataV, nil)
	return err
}

func (s *store) storeMetNo(input *metno.MetNoWeatherOutput) error {
	if input.Product == nil || input.Product.Time == nil {
		return nil
	}
	log := Log()
	for _, v := range input.Product.Time {
		if v.Location == nil || v.Location.Temperature == nil {
			continue

		}
		value := &pb.WeatherStoreValue{
			UpdatedTimestamp: now(),
			From:             uint32(v.From.Unix()),
			To:               uint32(v.To.Unix()),
			Lat:              v.Location.Latitude,
			Lng:              v.Location.Longitude,
		}

		if v.Location.Humidity != nil {
			value.Humidity = &pb.Unit{
				Unit:  v.Location.Humidity.Unit,
				Value: v.Location.Humidity.Value,
			}
		}

		if v.Location.Fog != nil {
			value.Fog = &pb.Percent{
				Id:      v.Location.Fog.ID,
				Percent: v.Location.Fog.Percent,
			}
		}

		if v.Location.Cloudiness != nil {
			value.Cloudiness = &pb.Percent{
				Id:      v.Location.Cloudiness.ID,
				Percent: v.Location.Cloudiness.Percent,
			}
		}

		if v.Location.LowClouds != nil {
			value.LowClouds = &pb.Percent{
				Id:      v.Location.LowClouds.ID,
				Percent: v.Location.LowClouds.Percent,
			}
		}

		if v.Location.HighClouds != nil {
			value.HighClouds = &pb.Percent{
				Id:      v.Location.HighClouds.ID,
				Percent: v.Location.HighClouds.Percent,
			}
		}

		if v.Location.MediumClouds != nil {
			value.MediumClouds = &pb.Percent{
				Id:      v.Location.MediumClouds.ID,
				Percent: v.Location.MediumClouds.Percent,
			}
		}

		if v.Location.WindSpeed != nil {
			value.WindSpeed = &pb.Speed{
				Id:       v.Location.WindSpeed.ID,
				Mps:      v.Location.WindSpeed.Mps,
				Name:     v.Location.WindSpeed.Name,
				Beaufort: v.Location.WindSpeed.Beaufort,
			}
		}

		if v.Location.WindGust != nil {
			value.WindGust = &pb.Speed{
				Id:  v.Location.WindGust.ID,
				Mps: v.Location.WindGust.Mps,
			}
		}

		if v.Location.AreaMaxWindSpeed != nil {
			value.AreaMaxWindSpeed = &pb.Speed{
				Mps: v.Location.AreaMaxWindSpeed.Mps,
			}
		}

		if v.Location.WindDirection != nil {
			value.WindDirection = &pb.Deg{
				Id:   v.Location.WindDirection.ID,
				Name: v.Location.WindDirection.Name,
				Deg:  v.Location.WindDirection.Deg,
			}
		}

		if v.Location.Pressure != nil {
			value.Pressure = &pb.Unit{
				Id:    v.Location.Pressure.ID,
				Unit:  v.Location.Pressure.Unit,
				Value: v.Location.Pressure.Value,
			}
		}

		if v.Location.TemperatureProbability != nil {
			value.TemperatureProbability = &pb.Unit{
				Unit:  v.Location.TemperatureProbability.Unit,
				Value: v.Location.TemperatureProbability.Value,
			}
		}

		if v.Location.WindProbability != nil {
			value.WindProbability = &pb.Unit{
				Unit:  v.Location.WindProbability.Unit,
				Value: v.Location.WindProbability.Value,
			}
		}

		if v.Location.DewpointTemperature != nil {
			value.DewpointTemperature = &pb.Unit{
				Id:    v.Location.DewpointTemperature.ID,
				Unit:  v.Location.DewpointTemperature.Unit,
				Value: v.Location.DewpointTemperature.Value,
			}
		}

		value.Temperature = &pb.Unit{
			Id:    v.Location.Temperature.ID,
			Unit:  v.Location.Temperature.Unit,
			Value: v.Location.Temperature.Value,
		}

		key := &pb.WeatherStoreKey{
			Lat:       v.Location.Latitude,
			Lng:       v.Location.Longitude,
			Timestamp: closestHour(v.From),
		}

		err := s.setStoredWeather(key, value)
		if err != nil {
			return err
		}
		log.Infof("%#v temp: %.2f", key, value.Temperature.Value)
	}
	return nil

}

func (s *store) updateTheWorld() error {
	client := metno.SimpleClient(10)
	log := Log()
	for i := 0; i < 180; i++ {
		for j := 0; j < 360; j++ {
			lat := float64(i - 90)
			if lat == -90 {
				lat = 0
			}
			lng := float64(j - 180)
			if lng == -180 {
				lng = 0
			}
			out, err := metno.LocationForecast(client, lat, lng, 0)
			if err != nil {
				log.Infof("failed to get data for %.2f/%.2f %s", lat, lng, err.Error())
				continue
			}
			err = s.storeMetNo(out)
			if err != nil {
				log.Infof("failed to store data for %.2f/%.2f %s", lat, lng, err.Error())
				return err
			}
			time.Sleep(2 * time.Second)
		}
	}
	return nil
}