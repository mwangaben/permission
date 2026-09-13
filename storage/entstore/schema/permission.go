package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Permission struct{ ent.Schema }

func (Permission) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "permissions"},
	}
}

func (Permission) Fields() []ent.Field {
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

func (Permission) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name", "guard_name", "tenant_id").Unique().
			StorageKey("idx_permissions_name_guard_tenant"),
		index.Fields("tenant_id").StorageKey("idx_permissions_tenant"),
		index.Fields("deleted_at").StorageKey("idx_permissions_deleted_at"),
	}
}
