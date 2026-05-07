const mongoHost = process.env.SAMPLE_AND_DIAGNOSIS_API_MONGODB_HOST
const mongoPort = process.env.SAMPLE_AND_DIAGNOSIS_API_MONGODB_PORT

const mongoUser = process.env.SAMPLE_AND_DIAGNOSIS_API_MONGODB_USERNAME
const mongoPassword = process.env.SAMPLE_AND_DIAGNOSIS_API_MONGODB_PASSWORD

const database = process.env.SAMPLE_AND_DIAGNOSIS_API_MONGODB_DATABASE
const collection = process.env.SAMPLE_AND_DIAGNOSIS_API_MONGODB_COLLECTION

const retrySeconds = parseInt(process.env.RETRY_CONNECTION_SECONDS || "5") || 5;

function connectionUri() {
  if (mongoUser) {
    return `mongodb://${mongoUser}:${mongoPassword}@${mongoHost}:${mongoPort}`;
  }
  return `mongodb://${mongoHost}:${mongoPort}`;
}

let connection;
while (true) {
  try {
    connection = Mongo(connectionUri());
    break;
  } catch (exception) {
    print(`Cannot connect to mongoDB: ${exception}`);
    print(`Will retry after ${retrySeconds} seconds`);
    sleep(retrySeconds * 1000);
  }
}

const db = connection.getDB(database);

if (!db.getCollectionNames().includes(collection)) {
  db.createCollection(collection);
}

db[collection].createIndex({ "id": 1 });
db[collection].createIndex({ "sampleCode": 1 }, { unique: true });
db[collection].createIndex({ "patientId": 1 });
db[collection].createIndex({ "status": 1 });
db[collection].createIndex({ "testTypes": 1 });

process.exit(0);
