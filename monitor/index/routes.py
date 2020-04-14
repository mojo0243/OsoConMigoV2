import time, os, datetime, cmd, logging
from datetime import date
from flask import render_template, url_for, flash, redirect, request, abort, make_response, jsonify, send_file
from index.login import newUser, loginForm
from index.models import users, clients, tasks, results
from index import app, db, bcrypt
from flask_login import login_user, current_user, logout_user, login_required

# Create Custom Logs for Node Check Ins

@app.after_request
def log_file_format(response):
	if request.path == '/':
		return response
	elif request.path == '/login':
		return response
	elif request.path == '/create-user':
		return response
	elif request.path == "/logout":
		return response
	elif request.path.startswith('/static'):
		return response
	elif request.path.startswith('/js'):
		return response
	else:
		return about(404)

# Define responses for requested URLs

@app.errorhandler(404)
def not_found(error):
	return make_response(jsonify({'error': 'Not found'}), 404)

@app.route("/", methods=['GET', 'POST'])
@login_required
def index():
        tgts = clients.query.all()
        alrts = clients.query.order_by(clients.lastseen.desc()).limit(8).all()
        jbs = tasks.query.order_by(tasks.id.desc()).limit(20).all()
        return render_template('main.html', tgts=tgts, alrts=alrts, jbs=jbs)

@app.route("/create-user", methods=['GET', 'POST'])
#@login_required
def createUser():
#	if current_user.username == 'admin':
	form = newUser()
	if form.validate_on_submit():
		hashed_pw = bcrypt.generate_password_hash(form.password.data).decode('utf-8')
		user = users(username=form.username.data, password=hashed_pw)
		db.session.add(user)
		db.session.commit()
		flash(f'User successfully created - {form.username.data}!', 'success')
		return redirect(url_for('createUser'))
	return render_template('user.html', form=form)
#	else:
#		return redirect(url_for('index'))

@app.route("/login", methods=['GET', 'POST'])
def signIn():
	if current_user.is_authenticated:
		return redirect(url_for('index'))
	form = loginForm()
	if form.validate_on_submit():
		user = users.query.filter_by(username=form.username.data).first()
		if user and bcrypt.check_password_hash(user.password, form.password.data):
			login_user(user, remember=form.remember.data)
			if user == users.query.first():
				return redirect(url_for('createUser'))
			else:
				return redirect(url_for('index'))
		else:
			flash(f'Failue to login, Username or Password is incorrect', 'danger')
	return render_template('login.html', form=form)

@app.route("/logout")
def logout():
	logout_user()
	return redirect(url_for('signIn'))
