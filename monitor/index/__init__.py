from flask import Flask
from flask_bcrypt import Bcrypt
from flask_sqlalchemy import SQLAlchemy
from flask_login import LoginManager

app = Flask(__name__)

# Define Database

app.config['SECRET_KEY'] = 'de3c2b39cfa44c1c8a9d98c06ab24cb3bd3f54d1d87d4f36d5cdffdd8980bbbb'

app.config['SQLALCHEMY_DATABASE_URI'] = 'postgresql://test:pass@localhost/oso'

db = SQLAlchemy(app)

bcrypt = Bcrypt(app)

login_manager = LoginManager(app)
login_manager.login_view = 'signIn'
login_manager.login_message_category = 'warning'

from index import routes
