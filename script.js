import http from 'k6/http';
import { check } from 'k6';
import { sleep } from 'k6';

// RUN WITH: k6 run --vus 10 --duration 10s script.js

/*
http.post("http://localhost:11000/key", JSON.stringify('{"foo": "bar"}'), { headers: { 'Content-Type': 'application/json' } });

export default function () {
  http.get('http://localhost:11000/key/user1');
  http.get('http://localhost:11001/key/user1');
  http.get('http://localhost:11002/key/user1');
  http.get('http://localhost:11003/key/user1');
  http.get('http://localhost:11004/key/user1');
  sleep(1);
}
*/

export default function () {

  /*
  let Get0 = {
    method: 'GET',
    url: 'http://localhost:11000/key/user1',
  };
  
  let req3 = {
    method: 'POST',
    url: 'https://httpbin.org/post',
    body: {
      hello: 'world!',
    },
    params: {
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    },
  };
  

  let responses = http.batch()
  */
  let responses = http.batch([
    ['GET', 'http://localhost:11000/key/user1', null, { tags: { ctype: 'application/json' } }],
    ['GET', 'http://localhost:11001/key/user1', null, { tags: { ctype: 'application/json' } }],
    ['GET', 'http://localhost:11002/key/user1', null, { tags: { ctype: 'application/json' } }],
    ['GET', 'http://localhost:11003/key/user1', null, { tags: { ctype: 'application/json' } }],
    ['GET', 'http://localhost:11004/key/user1', null, { tags: { ctype: 'application/json' } }],
  ]);
  check(responses[0], {
    'main page status was 200': (res) => res.status === 200,
  });
}