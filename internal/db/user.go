package db

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// User - A teamserver user.
type User struct {
	ID          uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt   time.Time `gorm:"->;<-:create;"`
	LastSeen    time.Time
	Name        string
	Token       string         `gorm:"uniqueIndex"`
	Permissions pq.StringArray `gorm:"type:text[]"`
}

// BeforeCreate - GORM hook.
func (o *User) BeforeCreate(tx *gorm.DB) (err error) {
	o.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}

	o.CreatedAt = time.Now()

	return nil
}

type Permissions struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`
	LastSeen  time.Time
	Name      string
}
