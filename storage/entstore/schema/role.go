package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Role struct{ ent.Schema }

func (Role) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "roles"},
	}
}

func (Role) Fields() []ent.Field {
	return []ent.Field{
		field.Uint("id").Unique().Immutable(),
		field.String("name").MaxLen(255).NotEmpty(),
		field.String("guard_name").MaxLen(100).Default("web"),
		field.String("tenant_id").MaxLen(100).Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("deleted_at").Optional().Nillable(),
	}
}

func (Role) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name", "guard_name", "tenant_id").Unique().
			StorageKey("idx_roles_name_guard_tenant"),
		index.Fields("tenant_id").StorageKey("idx_roles_tenant"),
		index.Fields("deleted_at").StorageKey("idx_roles_deleted_at"),
	}
}
