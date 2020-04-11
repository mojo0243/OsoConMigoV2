from datetime import datetime
from index import db, login_manager
from flask_login import UserMixin

@login_manager.user_loader
def load_user(users_id):
	return users.query.get(int(users_id))

# Create Database structure

class users(db.Model, UserMixin):
	id = db.Column(db.Integer, primary_key=True)
	username = db.Column(db.String(20), unique=True, nullable=False)
	password = db.Column(db.String(60), nullable=False)

	def __repr__(self):
		return f"users('{self.username}')"

class clients(db.Model):
        id = db.Column(db.Integer, primary_key=True)
        node = db.Column(db.Text, unique=True)
        arch = db.Column(db.Text)
        os = db.Column(db.Text)
        secret = db.Column(db.Text)
        comms = db.Column(db.Integer)
        flex = db.Column(db.Integer)
        firstseen = db.Column(db.Integer)
        lastseen = db.Column(db.Integer)

class tasks(db.Model):
        id = db.Column(db.Integer, primary_key=True)
        node = db.Column(db.Text)
        job = db.Column(db.Integer, unique=True)
        command = db.Column(db.Text)
        status = db.Column(db.Text)
        taskdate = db.Column(db.Integer)
        completedate = db.Column(db.Integer)
        complete = db.Column(db.Boolean)

class results(db.Model):
        id = db.Column(db.Integer, primary_key=True)
        node = db.Column(db.Text)
        jobid = db.Column(db.Integer)
        output = db.Column(db.Text)
        completedate = db.Column(db.Integer)

