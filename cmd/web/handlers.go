package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"memorycards.eegurt.net/internal/models"
	"memorycards.eegurt.net/internal/validator"

	"github.com/julienschmidt/httprouter"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	cardSets, err := app.cardSets.ListAll(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	data := app.newTemplateData(r)
	data.CardSets = cardSets

	app.render(w, http.StatusOK, "home.html", data)
}

func (app *application) cardSetView(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil || id < 1 {
		app.notFound(w)
		return
	}

	cardSet, err := app.cardSets.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
		} else {
			app.serverError(w, err)
		}
		return
	}

	data := app.newTemplateData(r)
	data.CardSet = cardSet

	app.render(w, http.StatusOK, "view.html", data)
}

type cardSetCreateForm struct {
	Title               string `form:"title"`
	CardsNumber         int    `form:"cards_number"`
	validator.Validator `form:"-"`
}

func (app *application) cardSetCreate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	data.Form = cardSetCreateForm{}

	app.render(w, http.StatusOK, "create.html", data)
}

func (app *application) cardSetCreatePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
	}

	var form cardSetCreateForm

	err = app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Title), "title", "This field cannot be blank")
	form.CheckField(validator.MaxChars(form.Title, 100), "title", "This field cannot be more than 100 characters long")
	form.CheckField(validator.PermittedIntRange(form.CardsNumber, 3, 10), "cards_number", "This field must be at range from 3 to 10")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "create.html", data)
		return
	}

	id, err := app.cardSets.Insert(r.Context(), form.Title, form.CardsNumber)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Put(r.Context(), "id", id)
	app.sessionManager.Put(r.Context(), "title", form.Title)
	app.sessionManager.Put(r.Context(), "cards-number", form.CardsNumber)
	app.sessionManager.Put(r.Context(), "flash", "Empty card set successfully created!")

	http.Redirect(w, r, fmt.Sprintf("/cards/create/%d", id), http.StatusSeeOther)
}

type cardCreateForm struct {
	Question            string `form:"question"`
	Answer              string `form:"answer"`
	CardsNumber         int    `form:"-"`
	validator.Validator `form:"-"`
}

func (app *application) cardsCreate(w http.ResponseWriter, r *http.Request) {
	id := app.sessionManager.GetInt(r.Context(), "id")
	title := app.sessionManager.PopString(r.Context(), "title")
	cardsNumber := app.sessionManager.GetInt(r.Context(), "cards-number")

	data := app.newTemplateData(r)
	data.CardSet = app.cardSets.New(id, title)
	data.Form = make([]cardCreateForm, cardsNumber)

	app.render(w, http.StatusOK, "create_cards.html", data)
}

func (app *application) cardsCreatePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
	}

	id := app.sessionManager.PopInt(r.Context(), "id")
	cardsNumber := app.sessionManager.PopInt(r.Context(), "cards-number")
	var forms = make([]cardCreateForm, cardsNumber)

	err = app.decodePostForm(r, &forms)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	for _, form := range forms {
		form.CheckField(validator.NotBlank(form.Question), "question", "This field cannot be blank")
		form.CheckField(validator.NotBlank(form.Answer), "answer", "This field cannot be blank")
	}

	for _, form := range forms {
		if !form.Valid() {
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "create.html", data)
			return
		}
	}

	for _, form := range forms {
		err := app.cards.Insert(r.Context(), id, form.Question, form.Answer)
		if err != nil {
			app.serverError(w, err)
			return
		}
	}

	app.sessionManager.Put(r.Context(), "flash", "Card set successfully created!")

	http.Redirect(w, r, fmt.Sprintf("/cardset/view/%d", id), http.StatusSeeOther)
}

type userSignupForm struct {
	Name                string `form:"name"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userSignupForm{}
	app.render(w, http.StatusOK, "signup.html", data)
}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	var form userSignupForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
	form.CheckField(validator.MinChars(form.Password, 8), "password", "This field must be at least 8 characters long")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "signup.html", data)
		return
	}

	err = app.users.Insert(r.Context(), form.Name, form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			form.AddFieldError("email", "Email address is already in use")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "signup.html", data)
		} else {
			app.serverError(w, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Your signup was successful. Please log in.")
	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

type userLoginForm struct {
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userLoginForm{}
	app.render(w, http.StatusOK, "login.html", data)
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	var form userLoginForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "login.html", data)
		return
	}

	id, err := app.users.Authenticate(r.Context(), form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password is incorrect")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "login.html", data)
		} else {
			app.serverError(w, err)
		}
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)

	http.Redirect(w, r, "/cardset/create", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Remove(r.Context(), "authenticatedUserID")

	app.sessionManager.Put(r.Context(), "flash", "You've been logged out successfully!")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
