import argparse
from index import app

# Define variables arguments for the program

web_parser = argparse.ArgumentParser()

# Define variable arguments for Web Server

web_parser.add_argument('-p', '--port', dest='port', action='store', help='This flag will start the web server on this port')
web_parser.add_argument('-i', dest='ip', action='store', help='Provide a portrange to scan with a dash EX: 1-100')

server = web_parser.parse_args()

if __name__ == '__main__':
	app.run(debug=True, host=server.ip, port=server.port)
