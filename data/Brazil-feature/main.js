const axios = require("axios");
const cheerio = require("cheerio");
const puppeteer = require("puppeteer");
const { sleep } = require("sleep");

(async function main() {
  const url = "http://estacoes.charlier.com.br";

  try {
    const browser = await puppeteer.launch({ headless: false });
    const page = await browser.newPage();
    await page.setUserAgent(
      "Mozilla/5.0 (iPad; CPU OS 11_0 like Mac OS X) AppleWebKit/604.1.34 (KHTML, like Gecko) Version/11.0 Mobile/15A5341f Safari/604.1"
    );
    await page.goto(url);
    await page.waitForSelector("li");
    const html = await page.content();

    const $ = await cheerio.load(html);

    const listItems = $("#centerbox #contenedoire #estacoes")
      .find("[rel='modal']")
      .toArray();

    let weatherInBrazil = [];

    for (let item of listItems) {
      const station =
        item.children[1].children[1].children[1].children[0].children[0].data;

      // const coords = await getCoordinates(station);

      const temperatureC =
        item.children[1].children[3].children[1].children[3].children[0].data;
      const weatherDetails = item.children[3].children;
      const tempMax = weatherDetails[1].children[3].children[0].data;
      const tempMin = weatherDetails[3].children[3].children[0].data;
      const humidity = weatherDetails[5].children[3].children[0].data;
      const dewPoint = weatherDetails[7].children[3].children[0].data;
      const precipitation = weatherDetails[9].children[3].children[0].data;
      const pressureHPA = weatherDetails[11].children[3].children[0].data;
      const lastUpdated = weatherDetails[13].children[3].children[0].data;
      const windSpeed =
        item.children[5].children[1].children[3].children[1].children[0].data;
      const windDir = item.children[5].children[1].children[1].attribs.style; // rm or http://snowfence.cfans.umn.edu/Components/winddirectionanddegreeswithouttable3.htm

      weatherInBrazil.push({
        location: {
          station,
          lat: "coords.lat",
          lng: "coords.lng"
        },
        weather: {
          temperatureC,
          tempMax,
          tempMin,
          dewPoint,
          humidity,
          precipitation,
          pressureHPA,
          windSpeed,
          windDir,
          lastUpdated
        }
      });
    }

    sleep(2);

    debugger;
    browser.close();
  } catch (e) {
    console.log(e.message);

    console.log(`res: ${e.response}`);
  }
})();

async function getCoordinates(cityName) {
  const KEY = "3qrtTDxHwEEnR3z32ztw1tSZHVBcnK5m";
  const API_URL = `http://www.mapquestapi.com/geocoding/v1/address?key=${KEY}&location=${cityName},BR`;
  const response = await axios.get(API_URL);
  try {
    const data = response.data;
    const latLng = data.results[0].locations[0].latLng;

    console.log(latLng);
    return latLng;
  } catch (e) {
    console.log(e.message);
    console.log(`error, code: ${e.response.status}`);
  }
}
