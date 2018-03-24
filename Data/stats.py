import datetime
import matplotlib.pyplot as plt
from file_read_backwards import FileReadBackwards
import matplotlib.dates as mdates
import time

def main():
	plt.style.use('dark_background')
	count = 0
	firstLine = True
	timeStart = 0
	def parseTime(line):
		t2 = [*map(int, line[0].split("/"))]
		t = [*map(int, line[1].split(":"))]
		return datetime.datetime(*t2, *t)
		
	minutes = {}
	def initMinute(m):
		minutes[m] = {"ips": {}, "count_moves": {"/getMove": 0, "/getMoveSlow": 0}}

	def delta_t(t):
		return datetime.datetime.now() - t
		
	init_time = -1
	bin_size = 120
	with FileReadBackwards("G:\log.txt") as frb:
		for l in frb:
			splitted = [x for x in l.split(" ") if x != ""]
			t = parseTime(splitted)
			minute = delta_t(t).seconds//bin_size
			if(minute > (8*60*60)//bin_size):
				break
			if minute not in minutes:
				initMinute(minute)
			m = minute
			mO = minutes[m]
			
			#print(l[:100])
			if(len(splitted) < 4):
				print("Weird line:", splitted)
				break
			
			if splitted[2] == "GET":
				mO["count_moves"][splitted[3]] += 1
				ip = splitted[5].split(":")[0]
				if ip not in mO["ips"]:
					mO["ips"][ip] = 0
				mO["ips"][ip] += 1
				
	minuteSorted = sorted(minutes.keys())
	slow = []
	fast = []
	ips_c = []
	for minute in minuteSorted:
		slow.append(minutes[minute]["count_moves"]["/getMoveSlow"])
		fast.append(minutes[minute]["count_moves"]["/getMove"])
		ips_c.append(len(minutes[minute]["ips"]))

	plt.rcParams['axes.facecolor']='#323232'
	plt.rcParams['savefig.facecolor']='#323232'

	x_axis = [datetime.datetime.now() - datetime.timedelta(minutes=(i*bin_size)//60) for i in minuteSorted]
	plt.subplot(211)
	plt.title("LCPlay stats, UTC+1")
	plt.plot(x_axis, slow, label="Slow reqs count")
	plt.plot(x_axis, fast, label="Fast reqs count")
	plt.legend()
	ax = plt.subplot(212)
	plt.plot(x_axis, ips_c, label="Unique IPs")
	ax.xaxis_date()
	ax.xaxis.set_major_formatter(mdates.DateFormatter("%H:%M"))
	ax.xaxis.set_minor_formatter(mdates.DateFormatter("%H:%M"))

	plt.gcf().autofmt_xdate()
	plt.legend()
	plt.savefig("stats.png")

while True:
	main()
	time.sleep(120)