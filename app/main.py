import os

from flask import Flask, render_template, Response
from flask_cors import CORS, cross_origin
app = Flask(__name__)
cors = CORS(app)
app.config['CORS_HEADERS'] = 'Content-Type'

@app.route('/')
def home():
    return render_template('index.html')

@app.route('/file/<int:port>')
def file(port):
    with open(f'{port}.txt', 'r') as file:
        content = file.read()
    return Response(content, mimetype='text/plain')

if __name__ == '__main__':
    app.run(debug=True)