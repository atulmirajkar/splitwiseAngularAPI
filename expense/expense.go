package expense

import "time"

/*ExpensesWrapper  Wrapper to array of expenses*/
type ExpensesWrapper struct {
	Expenses []Expense `json:"expenses"`
}

/*Category type of expense*/
type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

/*UserInfo - User information*/
type UserInfo struct {
	UserID    int    `json:"user_id"`
	OwedShare string `json:"owed_share"`
}

/*Expense - a single expense*/
type Expense struct {
	ID          int        `json:"id"`
	GroupID     int        `json:"group_id"`
	Description string     `json:"description"`
	Date        time.Time  `json:"date"`
	Category    Category   `json:"category"`
	Users       []UserInfo `json:"users"`
}

/*ResponseExpense - a single expense with category*/
type ResponseExpense struct {
	Category  string    `json:"category"`
	UserID    int       `json:"user_id"`
	OwedShare string    `json:"owed_share"`
	Date      time.Time `json:"date"`
}

/********************************************User Structs*******************************/
type GroupWrapper struct {
	Group Group `json:"group"`
}

type Members struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
}

type Group struct {
	ID      int       `json:"ID"`
	Name    string    `json:"Name"`
	Members []Members `json:"members"`
}

type GroupArrWrapper struct {
	Groups []Group `json:"groups"`
}

/*******************************************categories********************************/

type CategoryWrapper struct {
	Categories []Categories `json:"categories"`
}
type Subcategories struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
type Categories struct {
	ID            int             `json:"id"`
	Name          string          `json:"name"`
	Subcategories []Subcategories `json:"subcategories"`
}
