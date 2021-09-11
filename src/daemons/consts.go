package daemons

const DEFAULT_CLUSTER = "local"
const DEFAULT_NUM_SERVERS = 10
const DEFAULT_NUM_READERS = 5
const DEFAULT_NUM_WRITERS = 5
const DEFAULT_NUM_SERVERS_PER_OBJECT = 5
const DEFAULT_FILE_SIZE = 10240
const DEFAULT_ALGORITHM = PRAKIS
const DEFAULT_OBJECT_PICK_DIST = "uniform" // or  "zipfian"
const DEFAULT_LOG_INTERVAL_STATS = 1       //seconds

const DEFAULT_NUM_OBJECTS = 10
const DEFAULT_WAITTIME_BTW_OPS_MS = 1
const DEFAULT_WAITTIME_BTW_READS_MS = 1  // Average Time assuming exponential waitng time
const DEFAULT_WAITTIME_BTW_WRITES_MS = 1 // Average Time assuming exponential waitng time
const DEFAULT_WRITETODISK = false
const DEFAULT_FOLDER_TO_STORE = "/tmp/ercost"
const NUM_PKTS_WRITE = 1000

const CHECK_SAFETY = false
const ENABLE_APPLICATION_LOGS = true // enable this for checking safety
const ENABLE_SYSTEM_LOGS = true
const ENABLE_DEBUG_LOGS = true
const ENBALE_EXP_LOGS = true
const APPLOGFILE = "applog.txt"
const SYSTEMLOGFILE = "systemlog.txt"
const DEBUGLOGFILE = "debuglog.txt"
const EXPLOGFILE = "experimentlog.txt"

const DEFAULT_MAXNUMOBJ = 10000
const DEFAULT_MAXOBJSIZEKB = 10000 // 10 MB
const DEFAULT_MINOBJSIZEKB = 1     // kB
// Erasure Coding Related Parameters

const CAUCHY = "Cauchy"
const DEFAULT_ERASURECODE_RATE = 0.6      // k/n, k = floor(rate*n)
const DEFAULT_ERASURECODE_METHOD = CAUCHY // we use ISAL for this method

// REAL constants, please do not change these
const SODAW = "SODAW"
const ABD = "ABD"
const ABD_FAST = "ABD_FAST"
const SODAW_FAST = "SODAW_FAST"
const PRAKIS = "SODA_WITH_LIST"
const READER = 0
const WRITER = 1
const SERVER = 2
const CONTROLLER = 3

const MAX_NUM_PENDING_READS = 1000000
const MAX_NUM_PENDING_WRITES = 1000000

const ALGO_PORT_IDX = 1
const HTTP_PORT_IDX = 2
const ALGO_PORT = "8081"
const HTTP_PORT = "8080"

const ABD_GET_TAG = "getTag"
const ABD_GET_DATA = "getData"
const ABD_PUT_DATA = "putData"

const SODAW_GET_TAG = "sodaw_getTag"
const SODAW_GET_DATA = "sodaw_getData"
const SODAW_PUT_DATA = "sodaw_putData"
const SODAW_READ_COMPLETE = "sodaw_readComplete"

const SODAW_FAST_GET_TAG_DATA = "sodaw_getTagData"

const PRAKIS_PUT_DATA = "prakis_put_data"
const PRAKIS_PUT_TAG = "prakis_put_tag"
const PRAKIS_GET_TAG_DATA = "prakis_get_tag_data"
const PRAKIS_RELAY_DATA = "prakis_relay_data"
const PRAKIS_READ_COMPLETE = "prakis_readComplete"
const PRAKIS_READ_COMMIT_TAG = "prakis_commit_tag_via_read"

const UNIFORM_OBJECT_PICK = "uniform"
const ZIPFIAN_OBJECT_PICK = "zipfian"

const YES = "YES"
const NO = "NO"

// ZMQ constants

const MAX_BACKLOG = 1000
const NON_BLOCKING = 1 // Put 0 for blocking

const TIMEOUT = 300
