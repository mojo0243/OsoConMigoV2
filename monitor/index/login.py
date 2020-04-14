from flask_wtf import FlaskForm
from wtforms import StringField, PasswordField, SubmitField, BooleanField, IntegerField
from wtforms.validators import DataRequired, Length, EqualTo, ValidationError
from index.models import users

class newUser(FlaskForm):
	username = StringField('Username', validators=[DataRequired(), Length(min=2, max=20)])
	password = PasswordField('Password', validators=[DataRequired(), Length(min=8)])
	confirmPassword = PasswordField('Confirm Password', validators=[DataRequired(), EqualTo('password')])
	submit = SubmitField('Create Account')

	def validate_username(self, username):
		user = users.query.filter_by(username=username.data).first()
		if user:
			raise ValidationError('A current user exists with that username')

class loginForm(FlaskForm):
	username = StringField('Username', validators=[DataRequired(), Length(min=2, max=20)])
	password = PasswordField('Password', validators=[DataRequired()])
	remember = BooleanField('Keep Me Logged In')
	submit = SubmitField('Login')
