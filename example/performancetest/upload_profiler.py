#!/usr/bin/python3
import re
import sys

FILE = "tmp.txt"
data = []
data_detail = []

stats = []
file_hash = []


def find_file_hash(file_path):
    with open(file_path) as f:
        for line in f:
            m = re.match('^file_upload_profiler.*checkpoint=.(v05.*):[RS][CN][VD]_',line)
            if m:
                exist = False
                for l in file_hash:
                    if l == m.group(1):
                        exist = True
                        break
                if exist == False:
                    file_hash.append(m.group(1))


def fetch_data(file_path):
    for fh in file_hash:
        with open(file_path) as f:
            for line in f:
                m = re.match('^file_upload_profiler.*checkpoint=.(v05.*):(..._.*)\"\} (1.6.*e\+15)',line)
                if m:
                    if m.group(1) == fh:
                        entry = []
                        entry.append(m.group(1))
                        entry.append(m.group(2))
                        entry.append(int(float(m.group(3))))
                        data.append(entry)
                m = re.match('^file_upload_profiler.*checkpoint=.(v05.*):(RCV_PROGRESS_DETAIL:.*)\"\} ([0-9]*)$',line)
                if m:
                    if m.group(1) == fh:
                        entry = []
                        entry.append(m.group(1))
                        entry.append(m.group(2))
                        entry.append(int(m.group(3)))
                        data_detail.append(entry)
    
def handle_data():
    global data
    data_tmp = []
    def keyfunc(elem):
        return elem[2]

    for fh in file_hash:
        dh = []
        for dh_line in data:
            if dh_line[0] == fh:
                dh.append(dh_line)
        dh.sort(key=keyfunc)
        # relative time
        initialTime = dh[0][2]
        for item in dh:
            item[2] = item[2] - initialTime
    
        for item in dh:
            ms = re.match('SND_FILE_DATA:([0-9]*)', item[1])
            if ms:
                for it in dh:
                    mr = re.match('RCV_PROGRESS:'+ms.group(1)+'$', it[1])
                    if mr:
                        it.append(int(it[2])-int(item[2]))
                        for it2 in data_detail:
                            md = re.match('RCV_PROGRESS_DETAIL:'+ms.group(1)+'$', it2[1])
                            if md:
                                it.append(it2[2])
        data_tmp.append(dh)
    data = data_tmp

def write_data_to_file(file_path):
    with open(file_path, 'w') as f:
        for i in range(len(data)):
            for j in range(len(data[i])):
                print(data[i][j], file=f)

def analyze_data():
    for i in range(len(data)):
        dh = data[i]
        stats = []
        stats.append(dh[-1][2])
        stats.append(dh[1][2])
        local_reading = 0
        for i in range(len(dh)):
            m = re.match("SND_GET_LOCAL_DATA:([0-9]*)", dh[i - 1][1])
            if m:
                for j in range(len(dh)):
                    m2 = re.match("RCV_GET_LOCAL_DATA:"+m.group(1), dh[j - 1][1])
                    if m2:
                        local_reading = local_reading + (dh[j - 1][2] - dh[i -1][2])
                        break
        stats.append(local_reading)

        storage_handle = 0
        for i in range(len(dh)):
            m = re.match("RCV_PROGRESS:", dh[i -1][1])
            if m:
                #print(data[i-1])
                storage_handle = storage_handle + (dh[i - 1][4])
        stats.append(storage_handle)

        starting_data = 0
        stopping_data = 0
        for i in range(len(dh)):
            m = re.match("SND_FILE_DATA:", dh[i - 1][1])
            if m:
                starting_data = dh[i - 1][2]
                break
        for i in range(len(dh)):
            m = re.match("RCV_PROGRESS:", dh[i - 1][1])
            if m:
                stopping_data = dh[i - 1][2]
        stats.append(stopping_data - starting_data)
        print(*stats, sep=',')
 

def find_hash_from_file(path):
    with open(path) as f:
        for line in f:
            line_list = line.split(',')
            exist = False
            for l in file_hash:
                if l == line_list[0]:
                    exist = True
                    break
            if exist == False:
                file_hash.append(line_list[0])
        
def read_data_from_file(path, hash):
    with open(path) as f:
        for line in f:
            l = line.split(',')
            if l[0] == hash:
                l.pop()
                for item in range(2, len(l)):
                    l[item] = int(l[item])
                data.append(l)


if __name__ == '__main__':
    if len(sys.argv) == 3:
        # read data from prom_log_file
        find_file_hash(sys.argv[1])
        fetch_data(sys.argv[1])

        # handle the log and output to handled_raw_file        
        handle_data()
        write_data_to_file(sys.argv[2])

        # analyze data and output to stdout in format of data_out
        print("total, file request, local read, storage handle, data transmit")
        analyze_data()

    else:
        print("usage:"+sys.argv[0]+" prom_log_file handled_raw_file")
